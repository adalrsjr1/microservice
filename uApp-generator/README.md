## Installation

Ensure that you have `pip` installed on your environment, than run:

`pip install -r requirements.txt`

## Running

`python3 uApp-generator.py`

The parameters are hardcoded. The script creates a directory named `generated/` where it will dump all of the manifests (deployment, service, searchspace, and  configmap) of each service, totalling 10 (number of services is hardcoded)

## Requirements

To show uApp tree it is necessary to install GraphViz-Dev.

More details in [pygraphviz](https://pygraphviz.github.io/documentation/pygraphviz-1.3rc1/install.html)
