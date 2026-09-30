"""Independent expectations for the real page's rendered fields and hierarchy."""
from html.parser import HTMLParser
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
class Page(HTMLParser):
    def __init__(self, source):
        super().__init__(convert_charrefs=True)
        self.stack=[]
        self.nodes=[]
        self.feed(source)
    def handle_starttag(self, tag, attrs):
        node={'tag':tag,'attrs':dict(attrs),'ancestors':self.stack.copy(),'text':''}
        self.nodes.append(node)
        if tag not in ['input','meta','br','hr','img','link']:
            self.stack.append(node)
    def handle_endtag(self, tag):
        assert self.stack and self.stack[-1]['tag']==tag, tag
        self.stack.pop()
    def handle_data(self, data):
        for node in self.stack: node['text']+=data

def verify(key, source):
    case=key.rsplit('_',1)[1]
    page=Page(source)
    assert not page.stack
    values={'blank':['','','','',''], 'saved':['21','logo','Logo','One logo','Design'], 'hostile':['1','logo','<script>','" onfocus="','</textarea>']}[case]
    forms=[n for n in page.nodes if n['tag']=='form']
    assert len(forms)==(1 if case=='blank' else 2)
    assert all(n['attrs']['method']=='post' and all(a['tag']!='form' for a in n['ancestors']) for n in forms)
    edit=next(n for n in forms if n['attrs']['action']=='/service/edit')
    controls=[n for n in page.nodes if edit in n['ancestors'] and n['tag'] in ['input','textarea']]
    expected=dict(zip(['id','slug','title','deliverable','description'],values))|{'csrf':'token'}
    assert {n['attrs']['name']:n['attrs'].get('value',n['text']) for n in controls}==expected
    for name,limit in [('slug','48'),('title','120'),('deliverable','500'),('description','4000')]:
        control=next(n for n in controls if n['attrs']['name']==name)
        assert control['attrs']['id']=='service_'+name and control['attrs']['maxlength']==limit
        assert len([n for n in page.nodes if n['tag']=='label' and n['attrs'].get('for')==control['attrs']['id']])==1
    if case!='blank':
        publication=next(n for n in forms if n is not edit)
        assert publication['attrs']['action']=='/service/state'
        assert {n['attrs']['name']:n['attrs']['value'] for n in page.nodes if publication in n['ancestors'] and n['tag']=='input'}=={'id':values[0],'csrf':'token','target_state':'published'}
    assert not any(n['tag']=='script' or any(k.startswith('on') for k in n['attrs']) for n in page.nodes)
    if '_edit_' in key:
        section=next(n for n in page.nodes if n['tag']=='section')
        assert [n['tag'] for n in page.nodes if n['ancestors'] and n['ancestors'][-1] is section]==['h1','p']
        if case!='blank': assert forms[0]['attrs']['action']=='/service/state'

rendered=json.loads((ROOT/'rendered.json').read_text())
for key, result in rendered.items(): verify(key,result['html'])
mutant=rendered['components_saved']['html'].replace('value="Logo"','value="One logo"')
try:
    verify('components_saved',mutant)
except AssertionError:
    mutation_rejected=True
else:
    raise AssertionError('wrong title binding was not caught')
result={'rendered_cases':len(rendered),'field_values_limits_labels_csrf_actions_methods_and_hierarchy':'pass','hostile_inputs_remain_data':'pass','synthetic_wrong_field_output_rejected':mutation_rejected,'scope':'HTMLParser and exact attribute/text expectations; no browser execution'}
(ROOT/'output-checks.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result))
