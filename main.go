package main

import (
	"flag"
	"fmt"
	"image"
	ic "image/color"
	"image/png"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"time"

	"strings"

	color "github.com/fatih/color"
	vnc "github.com/mitchellh/go-vnc"
	resize "github.com/nfnt/resize"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("Usage: vncscan <server>")
		return
	}

	address := flag.Args()[0]
	if strings.Index(address, ":") == -1 {
		address = net.JoinHostPort(address, "5900")
	}

	fmt.Println(color.YellowString("[+] Connecting to server %s....", address))

	nc, err := net.Dial("tcp", address)
	if err != nil {
		panic(err)
	}

	/*
		ioutil.ReadAll(nc)

		tlsconn := tls.Client(nc, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
			return
		}

		if err := tlsconn.Handshake(); err != nil {
			fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
			return
		}

		nc = tlsconn
	*/
	/*
		raw := vnc.PasswordAuth{Password: ""}

		fmt.Println("[+] Shaking hands....")
			err = raw.Handshake(nc)
			if err != nil {
				panic(err)
			}
	*/

	ch := make(chan vnc.ServerMessage)

	fmt.Println(color.YellowString("[+] Connected"))
	client, err := vnc.Client(nc, &vnc.ClientConfig{
		ServerMessageCh: ch,
		Exclusive:       false,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
		return
	}

	height := int(client.FrameBufferHeight)
	width := int(client.FrameBufferWidth)
	fmt.Println(height, width)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		fmt.Println(color.YellowString("[+] Waiting for messages...."))

		profileImage := image.NewRGBA(image.Rect(0, 0, width, height))

		for m := range ch {
			if fur, ok := m.(*vnc.FramebufferUpdateMessage); !ok {
				fmt.Printf("Got message: %s\n", reflect.TypeOf(m))
			} else {
				for _, r := range fur.Rectangles {
					if e, ok := r.Enc.(*vnc.RawEncoding); !ok {
					} else {
						fmt.Println(color.YellowString("[*] Framebuffer update received"))

						for y := int(0); y < int(r.Height); y++ {
							for x := int(0); x < int(r.Width); x++ {
								c := e.Colors[y*int(r.Width)+x]
								profileImage.SetRGBA(int(r.X)+x, int(r.Y)+y, ic.RGBA{
									R: uint8(c.R),
									G: uint8(c.G),
									B: uint8(c.B),
									A: 255,
								})
							}

						}

						filename := fmt.Sprintf("screenshot-%s.png", time.Now().Format("20060102150405"))
						if f, err := os.Create(filename); err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return

						} else if err = png.Encode(f, profileImage); err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						} else if err := f.Close(); err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						} else {

							fmt.Println(color.YellowString("[*] Screenshot written to %s...", filename))
						}

						fmt.Println(color.YellowString("[*] Upscaling image"))
						newImage := resize.Resize(1024*3, 768*3, profileImage, resize.Lanczos3)

						fmt.Println(color.YellowString("[*] OCR'ing frame buffer"))
						cmd := exec.Command("tesseract", "-", "-")
						stdin, err := cmd.StdinPipe()
						if err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						}

						cmd.Stdout = os.Stdout
						cmd.Stderr = ioutil.Discard

						if err = cmd.Start(); err != nil { //Use start, not run
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						}

						fmt.Println("==========================================")
						if err = png.Encode(stdin, newImage); err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						}

						stdin.Close()

						if err = cmd.Wait(); err != nil {
							fmt.Fprintln(os.Stderr, color.RedString("Error occured: %s", err.Error()))
							return
						}

						fmt.Println("==========================================")
						return
					}
				}
			}
		}
	}()

	fmt.Println(color.YellowString("[+] Sending update request..."))
	client.KeyEvent(65507, true)
	client.KeyEvent(65507, false)
	client.FramebufferUpdateRequest(false, 0, 0, 1024, 768)

	wg.Wait()

	fmt.Println(color.YellowString("[*] Finished"))
}
