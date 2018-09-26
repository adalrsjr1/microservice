from yaml import dump, load
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


if __name__=="__main__":
    g = Graph(1000, 42)
    dc = DockerCompose(g)
    dc.create('svc_', 'zipkin:9411', '100', '0.35', '100', '0')
    compose = open('test.yaml','w')
    dc.dump(out=compose)
    g.save()


    #yaml = YAML()
    #yaml.dump(svc, sys.stdout)


    #g = Graph(20, 42, False)
    #print(g.check())
    #g.draw()

    #h = Graph(20, 42, True)
    #print(h.check())
    #h.draw()

