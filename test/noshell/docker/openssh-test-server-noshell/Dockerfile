FROM wordpress

RUN apt update && apt install openssh-server -y

RUN useradd -rm -d /home/test -s /bin/bash test
RUN echo 'test:test' | chpasswd

# OpenSSH requires the parent directory and all directories above it to be owned by root
RUN chown root:root /var/www
RUN chmod 755 /var/www
# The public directory is owned by the user
RUN chown -R test:www-data /var/www/html

# Edit the sshd_config file to restrict access to one directory & disable shell access
RUN echo "Match User test" >> /etc/ssh/sshd_config
RUN echo "ForceCommand internal-sftp" >> /etc/ssh/sshd_config
RUN echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
RUN echo "ChrootDirectory /var/www" >> /etc/ssh/sshd_config
RUN echo "PermitTunnel no" >> /etc/ssh/sshd_config
RUN echo "AllowAgentForwarding no" >> /etc/ssh/sshd_config
RUN echo "AllowTcpForwarding no" >> /etc/ssh/sshd_config
RUN echo "X11Forwarding no" >> /etc/ssh/sshd_config

COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22

CMD ["/start.sh"]
