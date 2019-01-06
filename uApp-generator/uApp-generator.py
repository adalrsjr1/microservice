from yaml import dump, dump_all, load
import sys
import argparse
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

class Microservice:
    def __init__(self, name, children, isRoot=False, image='adalrsjr1/microservice'):
        self.name = name
        """ bytes """
        self.payload = 128          # bytes
        """ percentage """
        self.cpuLoad = 0.01         # percentage
        """ bytes """
        self.memoryRequest = 1024   # bytes
        """ milliseconds """
        self.processingTime = 100   # ms
        self.pidFilepath = '/tmp'
        """ list """
        self.children = children    # list
        """ ip:port """
        self.addr = ':8888'            # ip:port
        self.isRoot = isRoot
        self.image = image

    def dockerService(self, portToPublish='8888:8888'):
        strchildren = ' '.join(self.children)

        service = {
            'image': self.image,
            'container_name': self.name,
            'depends_on': self.children, 
            'command': '\
                --name=%s \
                --pid-path=%s \
                --payload=%s \
                --memory=%s \
                --processing-time=%s \
                --cpu-load=%s \
                --addr=%s \
                %s ' % (self.name, self.pidFilepath, self.payload, self.memoryRequest, self.processingTime, self.cpuLoad, self.addr, strchildren)
        }

        if self.isRoot:
            service['ports'] = [portToPublish]

        return service

    def kubernetesDeployment(self, namespace='uapp', replicas=1, nodename='', portToPublish='8888'):
        nodeSelector = {
            'beta.kubernetes.io/os': 'linux'
        }
        if bool(nodename):
            nodeSelector['kubernetes.io/hostname'] = nodename

        return {
            'apiVersion': 'apps/v1',
            'kind': 'Deployment',
            'metadata': {
                'name': self.name,
                'namespace': namespace,
                'labels': {'app': self.name}
            },
            'spec': {
                'replicas': replicas,
                'selector': {
                    'matchLabels': {
                        'app': self.name
                    }
                },
                'template': {
                    'metadata': {
                        'labels': {
                            'app': self.name
                        }
                    },
                    'spec': {
                        'containers': [{
                            'name': self.name,
                            'image': self.image,
                            'imagePullPolicy': 'Always',
                            'ports': [{'containerPort': int(portToPublish)}],
                            'args': [
                                '--name=%s' % self.name,
                                '--pid-path=%s' % self.pidFilepath,
                                '--payload=%s' % self.payload,
                                '--memory=%s' % self.memoryRequest,
                                '--processing-time=%s' % self.processingTime,
                                '--cpu-load=%s' % self.cpuLoad,
                                '--addr=%s' % self.addr
                            ] + self.children,
                        }],
                        'nodeSelector': nodeSelector
                    }
                }
            }
        }

class DockerCompose:
    def __init__(self, graph):
        self.g = graph
        self.scheme = {}

    def create(self, svc_prefix, payload=128, memoryRequest=1024, processingTime=100, cpuLoad=0.01):
        adjacency = g.adjacency()
        name_prefix = svc_prefix
        svcs = {}

        for key, value in adjacency.iteritems():
            children = [name_prefix+str(child) for child, v in value.iteritems()]

            microservice = Microservice(
                        name=name_prefix+str(key),
                        isRoot= True if key == 0 else False,
                        children=children)

            microservice.payload = payload
            microservice.memoryRequest = memoryRequest
            microservice.processingTime = processingTime
            microservice.cpuLoad = cpuLoad

            svcs[name_prefix+str(key)] = microservice.dockerService() 

        services = {}
        services['version'] = '3'
        services['services'] = svcs
        self.scheme = services

        return services

    def dump(self, out=sys.stdout):
        dump(self.scheme, out, tags=False, default_flow_style=False, encoding='utf8')

class Kubernetes:
    def __init__(self, graph):
        self.g = graph
        self.scheme = []

    def dump(self, out=sys.stdout):
        dump_all(self.scheme, out, tags=False, default_flow_style=False, encoding='utf8')

    def create(self, uappName, svc_prefix, nodename='', payload=128, memoryRequest=1024, processingTime=100, cpuLoad=0.01):
        yamlFiles = [self.namespace(uappName)]

        adjacency = g.adjacency()
        name_prefix = svc_prefix
        args = {}

        for key, value in adjacency.iteritems():
            children = [name_prefix+str(child)+'.'+uappName+'.svc.cluster.local' for child, v in value.iteritems()]
            name = name_prefix+str(key)

            microservice = Microservice(name, children, isRoot=not bool(key))
            microservice.payload = payload
            microservice.memoryRequest = memoryRequest
            microservice.processingTime = processingTime
            microservice.cpuLoad = cpuLoad

            yamlFiles.append(self.service(name, uappName, root=microservice.isRoot))

            yamlFiles.append(microservice.kubernetesDeployment(namespace=uappName, replicas=1))

        self.scheme = yamlFiles
        return yamlFiles

    def namespace(self, name):
        return {
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'metadata': {
                'name': name,
                'labels': {
                    'istio-injection': 'enabled'
                }
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
            'port': 8888,
            'targetPort': 8888,
            'protocol': 'TCP',
            'name': 'http'
        }

        if root:
            spec['type'] = 'NodePort'
            port['nodePort'] = int(externalport)

        spec['ports'] = [port]

        return spec

def runOnKubernetes(graph, nodes, payload, memory, processingTime, cpuLoad):
    k = Kubernetes(graph)
    k.create('uapp', 'svc-')
    kubernetes = open('k8s--%s.yaml' % (createName(nodes, payload, memory, processingTime, cpuLoad)), 'w')
    k.dump(out=kubernetes)

def runOnDocker(graph, nodes, payload, memory, processingTime, cpuLoad):
    dc = DockerCompose(graph)
    dc.create('svc_')
    compose = open('docker--%s' % (createName(nodes, payload, memory, processingTime, cpuLoad)),'w')
    dc.dump(out=compose)
    g.save()

def createName(nodes, payload, memory, processingTime, cpuLoad):
    return "app-nodes_%s-payload_%s-memory_%s-processingTime_%s-cpuLoad_%s" % (nodes, payload, memory, processingTime, cpuLoad)

if __name__=='__main__':
    # https://docs.python.org/3.3/library/argparse.html
    parser = argparse.ArgumentParser(description="Create deployment file")
    
    parser.add_argument('-n', '--nodes', help='number of instances -- default=10', type=int, default=10)
    parser.add_argument('-p', '--payload', help='size of message payload (bytes) -- default=128', type=int, default=128)
    parser.add_argument('-m','--memory', help='amount of message to be allocated for a mock message (bytes) -- default=128', type=int, default=128)
    parser.add_argument('-t','--processing-time', help='time to process a mock message (ms) -- default=1', type=int, default=1)
    parser.add_argument('-c','--cpu-load', help='cpu load to process a message (%%) -- default=0.1', type=float, default=0.1)
    parser.add_argument('-k','--kubernetes', help='if true ues k8s otherwise docker compose -- default=True', type=bool, default=True)
    

    args = parser.parse_args()

    g = Graph(args.nodes, 31)

    if args.kubernetes:
        runOnKubernetes(g, args.nodes, args.payload, args.memory, args.processing_time, args.cpu_load)
    else:
        runOnDocker(g, args.nodes, args.payload, args.memory, args.processing_time, args.cpu_load)

    exit(0)