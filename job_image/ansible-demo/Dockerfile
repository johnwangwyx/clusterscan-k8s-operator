FROM python:3.8-slim


WORKDIR /usr/src/app
RUN pip install --no-cache-dir ansible
COPY playbook.yml ./

# Run
ENTRYPOINT ["ansible-playbook", "playbook.yml"]
