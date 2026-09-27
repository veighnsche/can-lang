"""Bounded synthetic checks for dimensionless report ranking."""
import copy
import unittest

import ranking

SLICES = dict(zip(('compiler', 'assertions', 'artifacts', 'generated', 'runtime', 'codecs', 'startup', 'browser', 'server', 'io', 'editor', 'journeys'),
                  ('ns/op', 'ns/op', 'ns/op', 'ns/op', 'ns/op', 'ns/op', 'ms/launch', 'ms/interaction', 'ms/trial', 'ms/op', 'ns/op', 'ms/journey')))


def manifest():
    return dict(status='complete', quality='measurement', requested_suites=list(SLICES), completed_suites=list(SLICES), environment={'host': 'test'}, profile='quick', trials=2, iterations=1, warmups=0, size=4, isolation='exclusive', source={'revision': 'new', 'dependencies': 'same', 'harness_overlays': 'same'})


def summary():
    return {key + '/case': dict(unit=unit, parameters={'size': 4}, timing_scope='test', iterations_per_sample=1,
                                distribution=dict(median=(i + 1) * 100, n=2, median_absolute_deviation=1))
            for i, (key, unit) in enumerate(SLICES.items())}


def targets(m, s):
    result = ranking.target_template(m, s)
    for key, row in result['cases'].items():
        row.update(target=s[key]['distribution']['median'] / 2, rationale='Reviewed synthetic goal')
    return result


class RankingTests(unittest.TestCase):
    def setUp(self):
        self.m, self.s = manifest(), summary()
        self.t = targets(self.m, self.s)

    def board(self, **kwargs):
        return ranking.build_rankings(self.m, self.s, **kwargs)

    def test_all_twelve_ratio_order_and_no_truncation(self):
        self.assertEqual(ranking.SUITE_UNITS, SLICES)
        b = self.board(targets=self.t)['targets']
        self.assertEqual(b['status'], 'ranked')
        self.assertEqual(len(b['rows']), 12)
        self.assertEqual([r['case'] for r in b['rows']], sorted(self.s))
        self.assertTrue(all(r['ratio'] == 2 for r in b['rows']))
        self.assertEqual(b['eligible_cases'], 12)
        self.t['cases']['runtime/case']['target'] /= 2
        self.assertEqual(self.board(targets=self.t)['targets']['rows'][0]['case'], 'runtime/case')

    def test_template_no_observed_targets_and_revision_excluded(self):
        t = ranking.target_template(self.m, self.s)
        self.assertTrue(all(r['target'] is None and r['rationale'] == '' for r in t['cases'].values()))
        self.assertNotIn('revision', t['context']['source'])
        self.assertEqual(self.board(targets=t)['targets']['eligible_cases'], 0)

    def test_benefit_inversion_and_zero(self):
        key = next(iter(self.s))
        self.s[key]['unit'] = 'ops/s'
        self.t = targets(self.m, self.s)
        self.t['cases'][key]['target'] = 200
        self.assertEqual(self.board(targets=self.t)['targets']['per_case'][key]['ratio'], 2)
        self.s[key]['distribution']['median'] = 0
        self.assertIn('zero observed', self.board(targets=self.t)['targets']['exclusions'][key])

    def test_all_better_still_eligible(self):
        for r in self.t['cases'].values():
            r['target'] *= 4
        b = self.board(targets=self.t)['targets']
        self.assertEqual(b['rows'], [])
        self.assertEqual(b['status'], 'ranked')
        self.assertEqual(b['eligible_cases'], 12)

    def test_missing_unset_stale_contract(self):
        keys = list(self.s)
        del self.t['cases'][keys[0]]
        self.t['cases'][keys[1]]['target'] = None
        self.t['cases'][keys[2]]['parameters'] = {}
        self.t['cases']['old/case'] = copy.deepcopy(self.t['cases'][keys[3]])
        b = self.board(targets=self.t)['targets']
        self.assertEqual(b['status'], 'partial')
        self.assertEqual(b['eligible_cases'], 9)
        self.assertEqual(len(b['exclusions']), 4)

    def test_context_mismatch(self):
        self.t['context']['size'] = 100
        self.assertEqual(self.board(targets=self.t)['targets']['status'], 'unavailable')
        old = copy.deepcopy(self.m)
        old['environment'] = {}
        self.assertIn('context', self.board(baseline=(old, self.s))['baseline']['reason'])

    def test_invalid_targets(self):
        key = next(iter(self.s))
        for value in (0, -1, float('nan'), float('inf'), True, '2'):
            with self.subTest(value=value):
                self.t['cases'][key]['target'] = value
                with self.assertRaises(ValueError):
                    self.board(targets=self.t)
        self.t = targets(self.m, self.s)
        self.t['cases'][key]['rationale'] = ' '
        with self.assertRaises(ValueError):
            self.board(targets=self.t)
        with self.assertRaises(ValueError):
            self.board(targets=[])

    def test_bad_observation_and_unknown_unit(self):
        key = next(iter(self.s))
        for value in (float('nan'), True, -1):
            self.s[key]['distribution']['median'] = value
            self.assertIn(key, self.board(targets=self.t)['targets']['exclusions'])
        self.s[key]['distribution']['median'] = 0
        self.assertEqual(self.board(targets=self.t)['targets']['per_case'][key]['ratio'], 0)
        self.s[key]['unit'] = 'mystery'
        self.t = targets(self.m, self.s)
        self.t['cases'][key].update(target=1, rationale='review')
        self.assertIn('unknown', self.board(targets=self.t)['targets']['exclusions'][key])

    def test_candidate_and_baseline_admission(self):
        for field, value in [('quality', 'smoke'), ('quality', 'exploratory'), ('status', 'failed'), ('completed_suites', [])]:
            bad = copy.deepcopy(self.m)
            bad[field] = value
            self.assertEqual(ranking.build_rankings(bad, self.s, self.t)['targets']['status'], 'ineligible')
            self.assertEqual(self.board(baseline=(bad, self.s))['baseline']['status'], 'unavailable')

    def test_baseline_binding_zero_and_server_issues(self):
        old = copy.deepcopy(self.s)
        for row in old.values():
            row['distribution']['median'] /= 2
        b = self.board(baseline=(self.m, old))['baseline']
        self.assertEqual(len(b['rows']), 12)
        old['server/case']['ranking_issues'] = ['generator drops']
        self.s['io/case']['ranking_issues'] = ['bad accounting']
        old['runtime/case']['distribution']['median'] = 0
        b = self.board(baseline=(self.m, old))['baseline']
        self.assertEqual(b['eligible_cases'], 9)
        self.assertEqual(b['status'], 'partial')
        old['runtime/case']['parameters'] = {}
        self.assertIn('contracts', self.board(baseline=(self.m, old))['baseline']['reason'])
        del old['runtime/case']
        self.assertIn('inventory', self.board(baseline=(self.m, old))['baseline']['reason'])

    def test_strict_manifest_and_context(self):
        for mutate in (
            lambda m: m.pop('size'),
            lambda m: m.update(source=None),
            lambda m: m['source'].pop('dependencies'),
            lambda m: m.update(requested_suites=['runtime', 'runtime']),
            lambda m: m.update(requested_suites='runtime'),
            lambda m: m.update(requested_suites=[1]),
            lambda m: m.update(completed_suites=list(reversed(m['requested_suites']))),
        ):
            bad = copy.deepcopy(self.m)
            mutate(bad)
            self.assertEqual(ranking.build_rankings(bad, self.s)['targets']['status'], 'ineligible')
            with self.assertRaises(ValueError):
                ranking.target_template(bad, self.s)
        for mutate in (lambda c: c.pop('size'), lambda c: c.update(source=None),
                       lambda c: c['source'].pop('harness_overlays')):
            bad = copy.deepcopy(self.t)
            mutate(bad['context'])
            with self.assertRaises(ValueError):
                ranking.validate_targets(bad)
        ranking.validate_targets(self.t)

    def test_requested_order_and_overflow(self):
        old = copy.deepcopy(self.m)
        old['requested_suites'].reverse()
        old['completed_suites'].reverse()
        self.assertIn('order', self.board(baseline=(old, self.s))['baseline']['reason'])
        key = next(iter(self.s))
        self.s[key]['distribution']['median'] = 1e307
        self.t['cases'][key]['target'] = 1
        self.assertIn('nonfinite', self.board(targets=self.t)['targets']['exclusions'][key])

    def test_empty_unavailable_and_no_side_effects(self):
        before = copy.deepcopy((self.m, self.s, self.t))
        b = self.board()
        self.assertEqual(b['targets']['total_cases'], 12)
        self.assertEqual(b['targets']['eligible_cases'], 0)
        self.board(targets=self.t)
        self.assertEqual((self.m, self.s, self.t), before)
        empty = ranking.build_rankings(self.m, {}, ranking.target_template(self.m, {}))['targets']
        self.assertEqual((empty['status'], empty['eligible_cases'], empty['total_cases']), ('unavailable', 0, 0))


if __name__ == '__main__':
    unittest.main()
