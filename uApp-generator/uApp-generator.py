from yaml import dump, dump_all, load
import sys
import networkx as nx
import matplotlib.pyplot as plt
from networkx.drawing.nx_agraph import graphviz_layout
from collections import OrderedDict
import pprint


class Graph:
    def __init__(self, n_nodes, seed=31, star=False):
        self.seed = seed
        self.star = star
        self.g = self.__newgraph(n_nodes, seed, star)

    def __newgraph(self, n_nodes, seed, star):
        m = 2
        if star:
            m = n_nodes-1
        g = nx.barabasi_albert_graph(n_nodes, m, seed=seed)

        dag = nx.DiGraph()

        edges = nx.dfs_tree(g, 0).edges()

        dag.add_edges_from(edges)

        return dag

    def check(self):
        return nx.is_directed_acyclic_graph(self.g) and nx.is_tree(self.g)

    def edges(self):
        return nx.dfs_tree(self.g,0).edges()

    def nodes(self):
        return self.g.nodes()

    def adjacency(self):
        #return nx.adjacency_matrix(self.g)
        return nx.to_dict_of_dicts(self.g)

    def draw(self):
        labels = {}
        for idx, node in enumerate(self.g.nodes()):
            labels[node] = idx

        pos = graphviz_layout(self.g)
        nx.draw(self.g,pos, arrows=True)
        nx.draw_networkx_labels(self.g, pos, labels)
        plt.show()

    def save(self):
        labels = {}
        for idx, node in enumerate(self.g.nodes()):
            labels[node] = idx

        pos = graphviz_layout(self.g)
        nx.draw(self.g,pos, arrows=True)
        nx.draw_networkx_labels(self.g, pos, labels)
        plt.savefig('graph.pdf')


class DockerCompose:
    def __init__(self, graph):
        self.g = graph
        self.scheme = {}

    def create(self, svc_prefix, zipkin, msgsize, load, msgtime, mem):
        adjacency = g.adjacency()
        name_prefix = svc_prefix
        svcs = {}

        for key, value in adjacency.iteritems():
            childs = [name_prefix+str(child) for child, v in value.iteritems()]
            svcs[name_prefix+str(key)] = self.__createservice(
                        name_prefix+str(key),
                        zipkin,
                        msgsize,
                        load,
                        msgtime,
                        mem,
                        childs,
                        True if key == 0 else False
                    )

        services = {}
        services['version'] = '3'
        services['services'] = svcs
        self.scheme = services

        return services

    def __createservice(self, name, zipkin, msgsize, load, msgtime, mem, childs, isRoot=False):
        strchilds = ' '.join(childs)
        svc = {
            'image': 'adalrsjr1/microservice',
            'container_name': name,
            'command': '--name=%s --zipkin=%s --msg-size=%s --msg-time=%s --load=%s --mem=%s %s' % \
            (name, zipkin, msgsize, msgtime, load, mem, strchilds),
            'depends_on': childs
        }

        if isRoot:
            svc['ports'] = ['8080:8080']

        return svc

    def dump(self, out=sys.stdout):
        dump(self.scheme, out, tags=False, default_flow_style=False, encoding='utf8')

class Kubernetes:
    def __init__(self, graph):
        self.g = graph
        self.scheme = []

    def dump(self, out=sys.stdout):
        dump_all(self.scheme, out, tags=False, default_flow_style=False, encoding='utf8')

    def create(self, uappName, svc_prefix, msgsize, load, msgtime, mem, sampling, nodename=''):
        yamlFiles = [self.namespace(uappName)]

        adjacency = g.adjacency()
        name_prefix = svc_prefix
        args = {}

        for key, value in adjacency.iteritems():
            childs = [name_prefix+str(child)+'.'+uappName+'.svc.cluster.local' for child, v in value.iteritems()]
            name = name_prefix+str(key)
            args['name'] = name
            args['msgsize'] = msgsize
            args['load'] = load
            args['msgtime'] = msgtime
            args['mem'] = mem
            args['sampling'] = sampling
            args['childs'] = childs

            root=not bool(key)

            yamlFiles.append(self.service(name, uappName, root=root))
            yamlFiles.append(self.deployment(name, uappName, args, sampling=sampling, nodename=nodename))

        self.scheme = yamlFiles
        return yamlFiles

    def namespace(self, name):
        return {
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'metadata': {
                'name': name
            }
        }

    def service(self, name, namespace, root=False, externalPort='30001'):
        return {
            'kind': 'Service',
            'apiVersion': 'v1',
            'metadata': {
                'name': name,
                'namespace': namespace
            },
            'spec': self.__spec(root, name, externalPort),
        }

    def __spec(self, root, name, externalport) :
        spec = {
            'selector': {
                'app': name
            }
        }

        port = {
            'port': 8080,
            'targetPort': 8080,
            'protocol': 'TCP',
            'name': 'http'
        }

        if root:
            spec['type'] = 'NodePort'
            port['nodePort'] = int(externalport)

        spec['ports'] = [port]

        return spec

    def deployment(self, name, namespace, args, sampling=True, nodename=''):
        nodeSelector = {
            'beta.kubernetes.io/os': 'linux'
        }
        if bool(nodename):
            nodeSelector['kubernetes.io/hostname'] = nodename

        return {
            'apiVersion': 'apps/v1',
            'kind': 'Deployment',
            'metadata': {
                'name': name,
                'namespace': namespace,
                'labels': {'app': name}
            },
            'spec': {
                'replicas': 1,
                'selector': {
                    'matchLabels': {
                        'app': name
                    }
                },
                'template': {
                    'metadata': {
                        'labels': {
                            'app': name
                        }
                    },
                    'spec': {
                        'containers': [{
                            'name': name,
                            'image': 'adalrsjr1/microservice:latest',
                            'imagePullPolicy': 'Always',
                            'ports': [{'containerPort': 8080}],
                            'args': [
                                '--name=%s' % name,
                                '--zipkin=$(ZIPKIN):9411',
                                '--sampling=%s' % args['sampling'],
                                '--msg-size=%s' % args['msgsize'],
                                '--load=%s' % args['load'],
                                '--msg-time=%s' % args['msgtime'],
                                '--mem=%s' % args['mem']
                            ] + args['childs'],
                            'env': [
                                {'name': 'ZIPKIN',
                                'value': 'zipkin.zipkin.svc.cluster.local'}
                            ]
                        }],
                        'nodeSelector': nodeSelector
                    }
                }
            }
        }



if __name__=="__main__":
    g = Graph(10, 31)
    dc = DockerCompose(g)
    dc.create('svc_', 'zipkin:9411', '100', '0.35', '100', '0')
    compose = open('test.yaml','w')
    dc.dump(out=compose)
    g.save()

    k = Kubernetes(g)
    k.create('uapp', 'svc-', '100', '0.35','100','0',True)
    kubernetes = open('k8s.yaml', 'w')
    k.dump(out=kubernetes)

    #yaml = YAML()
    #yaml.dump(svc, sys.stdout)


    #g = Graph(20, 42, False)
    #print(g.check())
    #g.draw()

    #h = Graph(20, 42, True)
    #print(h.check())
    #h.draw()

