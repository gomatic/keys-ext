package service

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	strings "strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func readerFromArgs(in string) (io.Reader, error) {
	if in != "" {
		file, err := os.Open(in)
		if err != nil {
			return nil, err
		}
		return bufio.NewReader(file), nil
	}
	return bufio.NewReader(os.Stdin), nil
}

func writerFromArgs(out string) (io.Writer, error) {
	if out != "" {
		file, createErr := os.Create(out)
		if createErr != nil {
			return nil, createErr
		}
		return file, nil
	}
	return os.Stdout, nil
}

func writeAll(writer io.Writer, b []byte) error {
	n, writeErr := writer.Write(b)
	if writeErr != nil {
		return writeErr
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}

func fmtKeys(keys []*Key) {
	out := &bytes.Buffer{}
	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 1, ' ', 0)
	for _, key := range keys {
		fmtKey(w, key)
	}
	w.Flush()
	fmt.Print(out.String())
}

func fmtUsers(usrs []*User) string {
	out := []string{}
	for _, usr := range usrs {
		out = append(out, fmt.Sprintf("%s@%s", usr.Name, usr.Service))
	}
	return strings.Join(out, ",")
}

func fmtKey(w io.Writer, key *Key) {
	if key == nil {
		fmt.Fprintf(w, "∅\n")
		return
	}
	fmt.Fprintf(w, "%s\t%s\t%s\n", key.KID, fmtUsers(key.Users), key.Type.Emoji())
}

func fmtItems(items []*Item) {
	out := &bytes.Buffer{}
	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 1, ' ', 0)
	for _, item := range items {
		fmtItem(w, item)
	}
	w.Flush()
	fmt.Print(out.String())
}

func fmtItem(w io.Writer, item *Item) {
	if item == nil {
		return
	}
	fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Type)
}

func argString(c *cli.Context, name string, optional bool) (string, error) {
	val := c.String(name)
	if val != "" {
		if c.NArg() > 0 {
			return "", errors.Errorf("too many arguments")
		}
		return val, nil
	}

	args := c.Args().First()
	if args != "" {
		return args, nil
	}

	if optional {
		return "", nil
	}

	return "", errors.Errorf("no %s specified", name)
}
