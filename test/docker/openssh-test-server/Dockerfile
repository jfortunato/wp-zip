FROM mysql:8.0-debian

RUN apt update && apt install openssh-server -y

RUN useradd -rm -d /home/test -s /bin/bash test
RUN echo 'test:test' | chpasswd

COPY public /home/test/public

COPY ./start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22

CMD ["/start.sh"]
