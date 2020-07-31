package main

// Helpfull snippets:
// /data/bugaev/PROG/GO/POST/second.html

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
    "strconv"
    "net"
    "text/template"
    // "bytes"
    "os"
    "os/exec"
    "encoding/json"
    "github.com/rs/cors"
)

var SelectUploadFileHtmlTmpl *template.Template

func LimitUpload40MB(r *http.Request) {
    // Parse our multipart form, 10 << 22 specifies a maximum
    // upload of 40 MB files.
    r.ParseMultipartForm(10 << 22)
}

func LimitUpload320MB(r *http.Request) {
    // Parse our multipart form, 10 << 22 specifies a maximum
    // upload of 40 MB files.
    r.ParseMultipartForm(10 << 25)
}

func ShakyVidBytes(r *http.Request, s *MySess) ([]byte, bool) {
    // FormFile returns the first file for the given key `myFile`
    // it also returns the FileHeader so we can get the Filename,
    // the Header and the size of the file
    file, header, err := r.FormFile("myFile")
    if err != nil {
        fmt.Println("Error Retrieving the File")
        fmt.Println(err)
        return nil, false
    }
    defer file.Close() // In case reading to memory fails.
    s.shaky_vid_orig_name = header.Filename
    fmt.Printf("Uploaded File: %+v\n", header.Filename)
    fmt.Printf("File Size: %+v\n", header.Size)
    fmt.Printf("MIME Header: %+v\n", header.Header)


    // read all of the contents of our uploaded file into a
    // byte array
    fileBytes, err := ioutil.ReadAll(file)
    if err != nil {
        fmt.Println(err)
	return nil, false
    }
    return fileBytes, true
}

type MySess struct {
  ID string
  TmpDir string
  shaky_vid_orig_name string
  shaky_vid_full_path string
  stabi_vid string
  stabi_vid_full_path string
  transform_full_path string
}


func ArrLastStr(str []string) string {
  return str[len(str) - 1]
}

func (s *MySess) WorkDir2ID() *MySess {
  DirParts := strings.Split(s.TmpDir, "/")
  s.ID = ArrLastStr(DirParts)
  fmt.Println("Session ID: ", s.ID)
  return s
}


func (s *MySess) MkTmpDir () (*MySess, error) {
    var err error
    s.TmpDir, err = ioutil.TempDir("WORKDIR", "")
    fmt.Println("Temporary directory is:", s.TmpDir, "<--")
    if err != nil {
        fmt.Println(err)
        return s, err
    }
    return s, nil
}

func (s *MySess) ShakyVidHdd (fileBytes []byte) (*MySess, error) {
    // Create input file "shaky.mp4" within WORKDIR:
    s.shaky_vid_full_path = s.TmpDir + "/shaky.mp4"
    err := ioutil.WriteFile(s.shaky_vid_full_path, fileBytes, 0644)
    if err != nil {
      fmt.Println(err)
    }
    return s, err
}

func (s *MySess)  TmpDir2stabi_vid_full_path() (*MySess, error) {
    s.stabi_vid_full_path = s.TmpDir + "/" + s.stabi_vid
    // Mock output file: s.stabi_vid_full_path = "WORKDIR" + "/" + s.stabi_vid
    return s, nil
}

func (s *MySess) vid_anal() (*MySess, error) {

    ffmpeg_exec, _ := exec.LookPath( "ffmpeg" )
    s.transform_full_path = s.TmpDir + "/transforms.trf"

    cmd := &exec.Cmd {
      Path:  ffmpeg_exec,
      Args: []string{ ffmpeg_exec, "-i", s.shaky_vid_full_path, "-vf", "vidstabdetect=shakiness=10:accuracy=15:result=" + s.transform_full_path, "-f", "null", "-" },
      Stdout: os.Stdout,
      Stderr: os.Stdout,
    }

    if err := cmd.Run(); err != nil {
      fmt.Println( "Error in vid_anal: ", err );
      return nil, err
    }
    return s, nil
}


func (s *MySess) vid_stab() (*MySess, error) {

    ffmpeg_exec, _ := exec.LookPath( "ffmpeg" )
    cmd := &exec.Cmd {
      Path:  ffmpeg_exec,
      Args: []string{ ffmpeg_exec, "-i", s.shaky_vid_full_path, "-vf", "vidstabtransform=input=" + s.transform_full_path + ",unsharp=5:5:0.8:3:3:0.4", s.stabi_vid_full_path },
      Stdout: os.Stdout,
      Stderr: os.Stdout,
    }

    if err := cmd.Run(); err != nil {
      fmt.Println( "Error in vid_stab: ", err );
      return nil, err
    }
    return s, nil
}

func append_base_name(orig string, appendix string) string {
  orig_parts := strings.Split(orig, ".")
  suff := ArrLastStr(orig_parts)
  suff_len := len(suff)
  base_name := orig[0:len(orig) - suff_len - 1] + appendix + "." + suff
  return base_name
}



func (s *MySess) send_stabi_vid_to_client(w http.ResponseWriter, r *http.Request) (*MySess, error) {
// https://stackoverflow.com/questions/24116147/how-to-download-file-in-browser-from-go-server
   w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(append_base_name(s.shaky_vid_orig_name, "_stab")))
   w.Header().Set("Content-Type", "application/octet-stream")
   http.ServeFile(w, r, s.stabi_vid_full_path)
   return s, nil
}


// Based on: https://tutorialedge.net/golang/go-file-upload-tutorial/
// TODO: Why is w not a pointer?
func uploadFile(w http.ResponseWriter, r *http.Request) {
    fmt.Println("File Upload Endpoint Hit")

    var err error

    Sess := &MySess{
      // w: w,
      // r: r,
      stabi_vid: "stabi_dummy.mp4",
      // Plus uninitialized fields.
    }


    _, err = Sess.MkTmpDir(); if err != nil { fmt.Printf("Error returned by MkTmpDir: %s\n", err); return }
    Sess.TmpDir2stabi_vid_full_path()
    Sess.WorkDir2ID()
    LimitUpload320MB(r)
    fileBytes, ok := ShakyVidBytes(r, Sess); if !ok { return }
    _, err = Sess.ShakyVidHdd(fileBytes); if err != nil { return }
    _, err = Sess.vid_anal(); if err != nil { return }
    _, err = Sess.vid_stab(); if err != nil { return }
    _, err = Sess.send_stabi_vid_to_client(w, r); if err != nil { return }
}

func RequestedHostByClient(r *http.Request) string {
    return r.Host
}

func SelectUploadFile(w http.ResponseWriter, r *http.Request) {
    // https://coderwall.com/p/ns60fq/simply-output-go-html-template-execution-to-strings
    // var buf bytes.Buffer
    // err := SelectUploadFileHtmlTmpl.Execute(&buf, MyURL{AddrWithPortSeenByClient: RequestedHostByClient(r)}); if err != nil { panic(err) }
    // fmt.Fprintf(w, buf.String())
    err := SelectUploadFileHtmlTmpl.Execute(w, MyURL{AddrWithPortSeenByClient: RequestedHostByClient(r)}); if err != nil { panic(err) }
}

func setupRoutes() {
    AllowCORS := true

    if !AllowCORS {
        http.HandleFunc("/upload", uploadFile)
        http.HandleFunc("/status", status_handler)
        http.HandleFunc("/", SelectUploadFile)
        http.ListenAndServe(":8080", nil)
    } else
    {
        mux := http.NewServeMux()
        mux.HandleFunc("/upload", uploadFile)
        mux.HandleFunc("/status", status_handler)
        mux.HandleFunc("/", SelectUploadFile)
        // cors.Default() setup the middleware with default options being
        // all origins accepted with simple methods (GET, POST). See
        // documentation below for more options.
        handler := cors.Default().Handler(mux)
        http.ListenAndServe(":8080", handler)
    }
}

func ListAllHostIP() {
    ifaces, err := net.Interfaces()
    if err != nil { return }
    // handle err
    for _, i := range ifaces {
        addrs, err := i.Addrs()
        if err != nil { return }
        for _, addr := range addrs {
            var ip net.IP
            switch v := addr.(type) {
            case *net.IPNet:
                    ip = v.IP
            case *net.IPAddr:
                    ip = v.IP
            }
            fmt.Println(ip.String())
        }
    }
}


func FirstHostIP4() (string, bool) {
// https://gist.github.com/jniltinho/9787946
    addrs, err := net.InterfaceAddrs()
    if err != nil {
       os.Stderr.WriteString("Oops: " + err.Error() + "\n")
       os.Exit(1)
    }

    for _, a := range addrs {
        if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                // os.Stdout.WriteString(ipnet.IP.String() + "\n")
                return ipnet.IP.String(), true
             }
        }
    }
    return "none", false
}

type MyURL struct {
    AddrWithPortSeenByClient string
}

func NewSelectUploadFileHtmlTmpl() {
    var err error
    SelectUploadFileHtmlTmpl, err = template.New("SelectUploadFileHtml").Parse(
`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Document</title>
  </head>
  <body>
    <form
      enctype="multipart/form-data"
      action="http://{{.AddrWithPortSeenByClient}}/upload"
      method="post"
    >
      <input type="file" name="myFile" />
      <input type="submit" value="upload" />
    </form>
  </body>
</html>`)
    if err != nil { panic(err) }
}

type Status struct {
    SizeMB int
    IsDone   bool
}

func status_handler(w http.ResponseWriter, r *http.Request) {
// From: https://www.golangprograms.com/example-to-handle-get-and-post-request-in-golang.html
// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
    if err := r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
    }
    var IsDone bool = false
    ID, err := strconv.Atoi(r.FormValue("ID")); if err != nil { return }
// https://www.alexedwards.net/blog/golang-response-snippets
    StatusResp := Status{ID + 1, IsDone}
    js, err := json.Marshal(StatusResp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // fmt.Fprintf(w, "%d", ID + 1) // Works fine.
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
    fmt.Printf("status_handler responded with: %q\n", js)
    return
}

func main() {
    fmt.Println("Hello World")
    NewSelectUploadFileHtmlTmpl()
    setupRoutes()
}

