create database cat_db;

create TABLE users(
   id integer NOT NULL GENERATED ALWAYS AS IDENTITY primary key,
   username varchar(50) unique,
   password varchar(50),
   email  varchar(50)
);

CREATE TABLE cat_adopt_posts
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY primary key,
	author_id integer,
	title character varying(50),
	city character varying(50),
	area character varying(50),
    cat_name character varying(50) ,
    cat_age character varying(50),
	cat_personality character varying(50),
	cat_story character varying(500)
);