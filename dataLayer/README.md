This is the data layer.

How to Run
Make sure mariadb is installed and running:
sudo apt install mariadb-server

Create a User in MariaDB

Log in to your database as the root user from your terminal:
sudo mysql

Create the new user. IMPORTANT: Replace 'YourSecurePassword' with a real password:
CREATE USER 'dataLayer_admin'@'localhost' IDENTIFIED BY 'YourSecurePassword';

Grant the new user full permissions, but only on the content_db database:
GRANT ALL PRIVILEGES ON content_db.* TO 'dataLayer_admin'@'localhost';

Apply the changes and exit the database shell:
FLUSH PRIVILEGES;
EXIT;

  
Make sure you have the Go MySQL driver installed:
go get github.com/go-sql-driver/mysql.

Run the application from your terminal in that directory:
go run .

You can now access the user interface at http://localhost:8080 and the admin interface at http://localhost:8080/schema.
Open your web browser and navigate to the two interfaces to test them:

    User Interface: http://localhost:8080

        You should see the "User: Content Manager" page, probably showing the contlet table by default. You can try adding and deleting data here.

    Admin Interface: http://localhost:8080/schema

        You should see the "Admin: Schema Editor" page. You can try adding a new column to one of the tables (for example, add an author column to the contlet table) and see how the user interface automatically updates.


