FROM hvaghani221/jupyter-base:1.0.0

COPY requirements.txt /tmp/requirements.txt
RUN pip install  -r /tmp/requirements.txt

CMD ["start-notebook.sh", "--NotebookApp.token='ab17a9eb56a95a0bb5af1befa3772368339592c3192da431'", "--NotebookApp.disable_check_xsrf=True"]
# docker buildx build --platform linux/amd64,linux/arm64/v8 -t hvaghani221/kernel:1.0.0 -f Dockerfile .
