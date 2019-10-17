#/bin/bash
# executed on docker-image (aws task) startup

[ -z "$NODEID" ] && ./dcron -nodeid 1 || ./dcron -nodeid $NODEID

