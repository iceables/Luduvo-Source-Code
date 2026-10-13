# Buildoverse

As the first surge of popularity hit our website, we knew PHP wasn't gonna cut it. Not because PHP is a bad language, but also because PHP is a bad language (especially raw PHP)... while I do know about Laravel and CakePHP and all these other frameworks, I also hate working with them. They definitely are good for somebody and something, but that somebody isn't me and that something we don't think is Buildoverse... And, having prior experience (a lot, of prior experience) in GO and many Buildoverse components being already built in GO, we thought: "why not just... rewrite it in GO?". This brought us here!

GO is very fast, and it is also very easy. Not only is GO very easy, but Fiber (the library we picked) is also very easy and perfect for Buildoverse, since we are moving (sorta) from Express.JS (Fiber is quite similar to it).

---

## How to run

Just two commands will get you set up and running:

```bash
go mod tidy
go run main.go
```

That'll start up the website at `localhost:3000`, where you'll be able to see and access the current stage of the site!
