#!/bin/bash
yum update -y
yum install -y httpd
systemctl start httpd
systemctl enable httpd
echo "<h1>Hello from ${environment} environment!</h1>" > /var/www/html/index.html
echo "<p>Health check endpoint</p>" > /var/www/html/health
