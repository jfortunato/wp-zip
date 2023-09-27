FROM mysql:8.0-debian

RUN apt update && apt install openssh-server libapache2-mod-php7.4 php7.4-mysql -y

RUN useradd -rm -d /home/test -s /bin/bash test
RUN echo 'test:test' | chpasswd

RUN chown -R test:www-data /var/www/html
RUN rm /var/www/html/index.html
COPY public /var/www/html

COPY ./start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22
EXPOSE 80

CMD ["/start.sh"]
