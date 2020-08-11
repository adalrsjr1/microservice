from yaml import dump, dump_all, load
import sys
import networkx as nx
import matplotlib.pyplot as plt
from networkx.drawing.nx_agraph import graphviz_layout
from collections import OrderedDict
import pprint
import random
import copy
random.seed(42)
import json
import os



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
    
    def getPaths(self):
        paths = []
        leafNodes = [node for node in self.g.nodes() if self.g.out_degree(node)==0]
        for leafNode in leafNodes:
            newPaths = nx.all_simple_paths(self.g, 0, leafNode)
            for newPath in newPaths:
                paths.append(newPath)
        return paths

class DockerCompose:
    def __init__(self, graph):
        self.g = graph
        self.scheme = {}

    def create(self, svc_prefix, zipkin, msgsize, msgtime, x, y):
        adjacency = g.adjacency()
        name_prefix = svc_prefix
        svcs = {}

        for key, value in adjacency.items():
            childs = [name_prefix+str(child) for child, v in value.items()]
            svcs[name_prefix+str(key)] = self.__createservice(
                        name_prefix+str(key),
                        zipkin,
                        msgsize,
                        msgtime,
                        x,
                        y,
                        childs,
                        True if key == 0 else False
                    )

        services = {}
        services['version'] = '3'
        services['services'] = svcs
        self.scheme = services

        return services

    def __createservice(self, name, zipkin, msgsize, msgtime, x, y, childs, isRoot=False):
        strchilds = ' '.join(childs)
        svc = {
            'image': 'adalrsjr1/microservice',
            'container_name': name,
            'command': '--name=%s --zipkin=%s --msg-size=%s --msg-time=%s --x=%s --y=%s %s' % \
            (name, zipkin, msgsize, msgtime, x, y, strchilds),
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

    def create(self, kobject, uappName, svc_name, svc_num, msgsize, msgtime, sampling, nodename=''):
        # yamlFiles = [self.namespace(uappName)]
        yamlFiles = []

        adjacency = g.adjacency()
        args = {}

        namespace = 'uapp'
        for key, value in adjacency.items():
            if key != svc_num:
                continue

            name = svc_name + '-' + str(svc_num) + '-mock'
            childs = [f'svc-{child}-mock.{namespace}.svc.cluster.local' for child, v in value.items()]
            args['name'] = name
            args['msgsize'] = msgsize
            args['msgtime'] = msgtime
            args['sampling'] = sampling
            args['childs'] = childs
            root=not bool(key)

        if kobject == 'service':
            yamlFiles.append(self.service(name, uappName, root=root))
        elif kobject == 'deployment':
            yamlFiles.append(self.deployment(name, uappName, args, sampling=sampling, nodename=nodename))
        elif kobject == 'searchspace':
            yamlFiles.append(self.searchspace(name, uappName))

        self.scheme = yamlFiles
        return yamlFiles

    def namespace(self, name):
        return ''
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
                'namespace': namespace,
                'annotations': {'injection.smarttuning.ibm.com': 'true'}
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
                'labels': {'app': name},
                'annotations': {'injection.smarttuning.ibm.com': 'true'}
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
                                '--name=$(NAME)',
                                '--zipkin=$(ZIPKIN):9411',
                                '--sampling=%s' % args['sampling'],
                                '--msg-size=$(MSG_SIZE)',
                                '--msg-time=$(MSG_TIME)',
                                '--x=$(X_VALUE)',
                                '--y=$(Y_VALUE)',
                                '--a=$(A_VALUE)',
                                '--b=$(B_VALUE)',
                                '--c=$(C_VALUE)',
                                '--d=$(D_VALUE)',
                                '--e=$(E_VALUE)',
                                '--f=$(F_VALUE)',
                                '--g=$(G_VALUE)',
                                '--h=$(H_VALUE)'
                            ] + args['childs'],
                            'env': [
                                {'name': 'ZIPKIN',
                                'value': 'zipkin.default.svc.cluster.local'}
                            ]
                            ,
                            'envFrom': [
                              {'configMapRef': {
                                'name': name + '-configmap'}}
                            ]
                        }],
                        'nodeSelector': nodeSelector
                    }
                }
            }
        }
    
    def searchspace(self, name, namespace):
        return {
            'apiVersion': 'smarttuning.ibm.com/v1alpha2',
            'kind': 'SearchSpace',
            'metadata': {
                'name': name + '-searchspace'
            },
            'spec': {
                'deployment': '',
                'namespace': namespace,
                'service': '',
                'manifests': [
                    {
                        'name': name,
                        'type': 'deployment',
                        'name': name+'-configmap',
                        'type': 'configMap'
                    }
                ]
            },
            'data': {
                'name': '',
                'tunables': {
                    'number': [
                        {
                            'name': 'X_VALUE',
                            'lower': '-10',
                            'upper': '10',
                            'step': '1'
                        },
                        {
                            'name': 'Y_VALUE',
                            'lower': '-10',
                            'upper': '10',
                            'step': '1'
                        },
                        {
                            'name': 'A_VALUE',
                            'lower': '-4',
                            'upper': '4',
                        },
                        {
                            'name': 'B_VALUE',
                            'lower': '-250',
                            'upper': '250',
                        },
                        {
                            'name': 'C_VALUE',
                            'lower': '-10',
                            'upper': '10',
                        },
                        {
                            'name': 'D_VALUE',
                            'lower': '1e-15',
                            'upper': '1e15',
                            'step': '10'
                        },
                        {
                            'name': 'E_VALUE',
                            'lower': '-2.5',
                            'upper': '2.5',
                        },
                        {
                            # can be also (-5,-10)
                            'name': 'F_VALUE',
                            'lower': '5',
                            'upper': '10',
                        },
                        {
                            'name': 'G_VALUE',
                            'lower': '-3',
                            'upper': '3',
                        },
                        {
                            'name': 'H_VALUE',
                            'lower': '-25',
                            'upper': '25',
                        },
                    ]
                }
            }
        }

class ConfigMap:
    def __init__(self, graph):
        self.g = graph
        self.scheme = []

    def dump(self, out=sys.stdout):
        dump_all(self.scheme, out, tags=False, default_flow_style=False, encoding='utf8')

    def create(self, uappName, svc_name, svc_num, msgsize, msgtime, sampling, nodename=''):
        yamlFiles = []
        adjacency = g.adjacency()
        # args = {}
        #
        # for key, value in adjacency.items():
        #     childs = [svc_name+str(child)+'.'+uappName+'.svc.cluster.local' for child, v in value.items()]
        #     name = svc_name+str(key)

        yamlFiles.append(self.config_map(svc_name+'-'+str(svc_num)+'-mock', uappName, msgsize, msgtime, sampling))

        self.scheme = yamlFiles
        return yamlFiles

    def config_map(self, name, namespace, msgsize, msgtime, sampling):
        config_map_name = name + '-configmap'
        return {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'metadata': {
                'name': config_map_name,
                'namespace': namespace
            },
            'data': {
                'NAME': name,
                'MSG_TIME': str(msgtime),
                'MSG_SIZE': str(msgsize),
                'X_VALUE': str(random.randint(-10, 10)),
                'Y_VALUE': str(random.randint(-10, 10)),
                'A_VALUE': str(random.uniform(-4, 4)),
                'B_VALUE': str(random.uniform(-250, 250)),
                'C_VALUE': str(random.uniform(-10, 10)),
                'D_VALUE': str(random.uniform(1e-5, 1e5)),
                'E_VALUE': str(random.uniform(-2.5, 2.5)),
                'F_VALUE': str(random.uniform(5, 10) if random.choice([True, False]) else random.uniform(-10, -5)),
                'G_VALUE': str(random.uniform(-3, 3)),
                'H_VALUE': str(random.uniform(-25, 25))

            }
        }


def pathsToMap(paths):
    # Convert node numbers to service name, add empty string to indicate that it is the end of a path
    for path in paths:
        for i in range(len(path)):
            path[i] = "svc-" + str(path[i]) + "-mock"
        path.append("")
        
    # Construct routeMap
    routeMap = {}
    for idx, path in enumerate(paths):
        route = {}
        for i in range(len(path)-1):
            route[path[i]] = path[i+1]

        routeMap[str(idx)] = route
    return routeMap

if __name__=="__main__":

    numOfApps = 5
    appName = 'svc'

    g = Graph(numOfApps)

    # Creates the directory to dump all manifests
    try:
        os.mkdir(os.getcwd()+'/generated')
    except:
        pass
    else:
        print('manifest target directory generated/ was created')


    dc = DockerCompose(g)
    dc.create('svc_', 'zipkin:9411', '100', '100', '2', '3')
    compose = open('generated/dockercompose.yaml','w') # will overwrite file with same name
    dc.dump(out=compose)

    # Creates all K8 related manifest files
    objects = ['service', 'deployment', 'searchspace']
    for i in range(numOfApps):
        k = Kubernetes(g)
        for o in objects:
            k.create(o, 'uapp', appName, i, '100', '100', True)
            fileName = appName+'-'+str(i)+'-'+o

            kubernetes = open('generated/'+fileName+'.yaml', 'w')
            k.dump(out=kubernetes)
        
        c = ConfigMap(g)
        c.create('uapp', appName, i, random.randint(100, 200), random.randint(100, 200), True)
        fileName = appName+'-'+str(i)+'-configmap'
        configmap = open('generated/'+fileName+'.yaml', 'w')
        c.dump(out=configmap)


    #Turns paths into map
    paths = g.getPaths()
    routeMap = pathsToMap(paths)
    # Write this routemap to a variable in a go file
    with open('../routeMap.go', 'w') as outfile:
        outfile.write("package main\nvar generatedRouteMap = map[string]map[string]string")
        json.dump(routeMap, outfile)