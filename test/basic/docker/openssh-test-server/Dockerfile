FROM wordpress

RUN apt update && apt install openssh-server -y

RUN useradd -rm -d /home/test -s /bin/bash test
RUN echo 'test:test' | chpasswd

RUN chown -R test:www-data /var/www/html

COPY ./start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22

CMD ["/start.sh"]
