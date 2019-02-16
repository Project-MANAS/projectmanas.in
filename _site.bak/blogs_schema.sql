DROP TABLE IF EXISTS Blogs;
CREATE TABLE Blogs(
    ID INTEGER PRIMARY KEY AUTOINCREMENT,
    Title Text NOT NULL,
    Descri Text NOT NULL,
    Author Text NOT NULL,
    PubDate Text NOT NULL,
    Image Text NOT NULL
);