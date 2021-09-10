create database cat_db;

create TABLE users(
   id integer NOT NULL GENERATED ALWAYS AS IDENTITY primary key,
   username varchar(50) unique,
   password varchar(50),
   name  varchar(50)
);


CREATE TABLE cat_adopt_posts
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY primary key,
	author_id integer,
	title character varying,
	city character varying,
	area character varying,
    cat_name character varying ,
    cat_age character varying,
	cat_personality character varying,
	cat_story character varying,
    contact_info character varying,
    img_src bytea
);