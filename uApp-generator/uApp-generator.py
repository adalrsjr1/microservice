import networkx as nx
import matplotlib.pyplot as plt
from networkx.drawing.nx_agraph import graphviz_layout

def newgraph(n_nodes, seed=42, star=False):
    #g = nx.gnc_graph(n_nodes,seed=42)
    m = 2
    if star:
        m = n_nodes-1
    g = nx.barabasi_albert_graph(n_nodes, m, seed=seed)

    dag = nx.DiGraph()

    dag.add_edges_from(edges(g))

    return dag

def check(g):
    return nx.is_directed_acyclic_graph(g) and nx.is_tree(g)

def edges(g):
    return nx.dfs_tree(g,0).edges()

def nodes(g):
    return g.nodes()

def draw(g):
    labels = {}
    for idx, node in enumerate(g.nodes()):
        labels[node] = idx

# Need to create a layout when doing
# separate calls to draw nodes and edges
    pos = graphviz_layout(g)
    nx.draw(g,pos, arrows=True)
    nx.draw_networkx_labels(g, pos, labels)
    plt.show()

if __name__=="__main__":
    g = newgraph(20, star=True)
    print(check(g))
    draw(g)

