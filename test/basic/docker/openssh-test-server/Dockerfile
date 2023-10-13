FROM wordpress

RUN apt update && apt install openssh-server default-mysql-client -y

# Download wp-cli so we can install wordpress
RUN curl https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar -o /tmp/wp-cli.phar
RUN chmod +x /tmp/wp-cli.phar
RUN mv /tmp/wp-cli.phar /usr/local/bin/wp

RUN useradd -rm -d /home/test -s /bin/bash test
RUN echo 'test:test' | chpasswd

RUN chown -R test:www-data /var/www/html

COPY ./start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22

CMD ["/start.sh"]
