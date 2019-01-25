First, get the binary.

    go get -u github.com/tiehuis/linguist-color-checker

Next, get the linguist languages.yml file that we will be parsing. The program
searches the current directory for a file named languages.yml.

    wget https://raw.githubusercontent.com/github/linguist/master/lib/linguist/languages.yml

To render an html page of all languages closer than the default minimum threshold.

    linguist-color-checker -html

Available options are:

    Usage of ./linguist-color-checker:
      -html
            render output as html instead of plaintext
      -threshold float
            threshold for printing color differences (default 10)
      -yaml string
            location of language specification file (default "languages.yml")
