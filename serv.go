// vim: ts=4 sw=4 
// set makeprg=go\ build\ serv.go
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
	"sync"
)

// Globals:
const sess_id_start = len("download/") + 1
var MockUpload = true
var SelectUploadFileHtmlTmpl *template.Template
var ProgressHtmlTmpl  *template.Template
var ID2Sess map[string]*MySess = make(map[string]*MySess)
var ID2SessMux sync.Mutex

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
  transforms_full_path string
  is_done bool
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

func (s *MySess) MockMkTmpDir () (*MySess, error) {
    var err error
	s.TmpDir = "WORKDIR/12345"
	if yes, err := file_system_entry_exists(s.TmpDir); yes {
		return s, err
	}
    err = os.MkdirAll(s.TmpDir, 0777)
    fmt.Println("Temporary directory is:", s.TmpDir, "<--")
    if err != nil {
        fmt.Println(err)
        return s, err
    }
    return s, nil
}


func (s *MySess) MockShakyVidHdd () (*MySess, error) {
    // Create input file "shaky.mp4" within WORKDIR:
    s.shaky_vid_orig_name = "shaky.mp4"
    s.shaky_vid_full_path = s.TmpDir + "/shaky.mp4"
    s.stabi_vid_full_path = s.TmpDir + "/stabi.mp4"
	s.transforms_full_path = s.TmpDir + "/transforms.trf"

	if yes, _ := file_system_entry_exists(s.shaky_vid_full_path); !yes {
		// Create a link to a resource file:
		err1 := os.Link("RESOURCE/shaky.mp4", s.shaky_vid_full_path)
		if err1 != nil {return s, err1}
	}

	if yes, _ := file_system_entry_exists(s.transforms_full_path); !yes {
	// Create a link to a resource file:
		err1 := os.Link("RESOURCE/transforms.trf", s.transforms_full_path)
		if err1 != nil {return s, err1}
	}

	if yes, _ := file_system_entry_exists(s.stabi_vid_full_path); !yes {
	// Create a link to a resource file:
		err1 := os.Link("RESOURCE/stabi.mp4", s.stabi_vid_full_path)
		if err1 != nil {return s, err1}
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
    s.transforms_full_path = s.TmpDir + "/transforms.trf"

    cmd := &exec.Cmd {
      Path:  ffmpeg_exec,
      Args: []string{ ffmpeg_exec, "-i", s.shaky_vid_full_path, "-vf", "vidstabdetect=shakiness=10:accuracy=15:result=" + s.transforms_full_path, "-f", "null", "-" },
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
      Args: []string{ ffmpeg_exec, "-i", s.shaky_vid_full_path, "-vf", "vidstabtransform=input=" + s.transforms_full_path + ",unsharp=5:5:0.8:3:3:0.4", s.stabi_vid_full_path },
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

func ffmpeg(s *MySess) {
	var err error
    _, err = s.vid_anal(); if err != nil { return }
    _, err = s.vid_stab(); if err != nil { return }
	s.is_done = true
}

// Based on: https://tutorialedge.net/golang/go-file-upload-tutorial/
// TODO: Why is w not a pointer?
func uploadFile(w http.ResponseWriter, r *http.Request) {
    fmt.Println("File Upload Endpoint Hit")

    var err error

    Sess := &MySess{
      stabi_vid: "stabi.mp4",
      // Plus uninitialized fields.
    }


    _, err = Sess.MockMkTmpDir(); if err != nil { fmt.Printf("Error returned by MkTmpDir: %s\n", err); return }
    Sess.TmpDir2stabi_vid_full_path()
    Sess.WorkDir2ID()
    LimitUpload320MB(r)
	if ! MockUpload {
	// We don't simulate upload, do everything honestly:
		fileBytes, ok := ShakyVidBytes(r, Sess); if !ok { return }
		_, err = Sess.ShakyVidHdd(fileBytes); if err != nil { return }
	} else {
	// We simulate upload, all files are already on disk, but not necessarily in WORKDIR:
		_, err = Sess.MockShakyVidHdd(); if err != nil { return }
		Sess.is_done = true
	    Sess.ID = "12345"
	}

	ID2SessMux.Lock()
	ID2Sess[Sess.ID] = Sess
	ID2SessMux.Unlock()
	if ! MockUpload {
		go ffmpeg(Sess)
	}
	Sess.ProgressHtml(w)
}

func RequestedHostByClient(r *http.Request) string {
    return r.Host
}

func SelectUploadFile(w http.ResponseWriter, r *http.Request) {
    // https://coderwall.com/p/ns60fq/simply-output-go-html-template-execution-to-strings
    // var buf bytes.Buffer
    // err := SelectUploadFileHtmlTmpl.Execute(&buf, MyURL{AddrWithPortSeenByClient: RequestedHostByClient(r)}); if err != nil { panic(err) }
    // fmt.Fprintf(w, buf.String())
	fmt.Println("In SelectUploadFile")
    err := SelectUploadFileHtmlTmpl.Execute(w, MyURL{AddrWithPortSeenByClient: RequestedHostByClient(r)}); if err != nil { panic(err) }
}

func (s *MySess) ProgressHtml(w http.ResponseWriter) (*MySess, error) {
    err := ProgressHtmlTmpl.Execute(w, *s); if err != nil { panic(err) }
	return s, err
}

func setupRoutes() {
    AllowCORS := true

    if !AllowCORS {
        http.HandleFunc("/upload", uploadFile)
        http.HandleFunc("/status", status_handler)
        http.HandleFunc("/download/", download_stabi_handler_get)
        http.HandleFunc("/", SelectUploadFile)
        http.ListenAndServe(":8080", nil)
    } else
    {
        mux := http.NewServeMux()
        mux.HandleFunc("/upload", uploadFile)
        mux.HandleFunc("/status", status_handler)
        mux.HandleFunc("/download/", download_stabi_handler_get)
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


func NewProgressHtmlTmpl() {
    var err error
    ProgressHtmlTmpl, err = template.New("ProgressHtmlTmpl").Parse(
`<!DOCTYPE html>
<html>
    <head>
        <title>Example</title>
    </head>


    <body onload="my_on_load()">

       <script>
			var timer_inst
		   	var sess_id = {{.ID}}
		   	var TimeInterval = 1000

		   	var xmlhttp = new XMLHttpRequest();
		   	var url = "/status";

		   	// DO SOMETHING HERE ONCE SERVER RETURNS (POSSIBLY INCOMPLETE) PROGRESS REPORT:
			xmlhttp.onreadystatechange = function() {
           		if (this.readyState == 4 && this.status == 200) {
                	var dict = JSON.parse(this.responseText);
					// DO WHAT YOU WANTED ONCE YOU GOT THE VALID REPORT:
                   	// document.getElementById("id01").innerHTML = dict["stabi_file_size_mb"]
		            document.getElementById("id01").innerHTML = this.responseText
		   			// Check the status after TimeInterval milliseconds, idle before that.
		   			// Enable timer only after the successful transaction with the server.
					if (!Boolean(dict["is_done"])) {
					   timer_inst = setTimeout(timer_callback, TimeInterval)
					} else {
   					   document.getElementById("id02").innerHTML = '<a href="/download/{{.ID}}"> Your processed file</a>'
					}
				}
			};

           
           function timer_callback() {
               var IDVar = "ID="
               var PostPar = IDVar.concat(sess_id)
               xmlhttp.open("POST", url, true);
               xmlhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
               
               xmlhttp.send(PostPar);
           }

           function my_on_load() {
               timer_inst = setTimeout(timer_callback, TimeInterval)
           }
       </script>

       <div id="id01"> To be replaced. </div>
       <div id="id02">  </div>
    </body>
</html>`)
    if err != nil { panic(err) }
}

type OutputStatus struct {
	Transforms_file_size_mb	int		`json:"transforms_file_size_mb"`
	Stabi_file_size_mb		int		`json:"stabi_file_size_mb"`
	Is_done					bool	`json:"is_done"`
}

// func (s *MySess) serve_progress_html()

// https://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-exists
// exists returns whether the given file or directory exists
func file_system_entry_exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return false, err
}


func file_size_mb_zero_if_nonexist(path string) (int, bool) {
	// fmt.Println("BEFORE DOING STAT FOR: ",  path)
	info, err := os.Stat(path);
	// fmt.Println("AFTER DOING STAT FOR: ",  path)
	// It is not a file but a directory - it shouldn't have happened:

	if err != nil {
		// Doesn't exist is OK --- it may not be created yet.
		if os.IsNotExist(err) {
			// fmt.Println("FILE DOESN'T EXIST: ",  path)
	        return 0, true
		}
		// Otherwise, err is a problem:
		fmt.Println("ERROR: FILE HAS SOME PROBLEM: ",  path)
		return 0, false
	}

	if info.IsDir() {
		fmt.Println("ERROR: FILE IS A DIRECTORY: ",  path)
		return 0, false }

	// And finally, if file exists and no errors triggered:
	x := info.Size()
	// fmt.Println("FILE EXISTS: ",  path)
	return int(x / 1000000), true // Return Millions of Bytes.
}

func (s *MySess) output_files_status() (*OutputStatus, bool) {
	stat := &OutputStatus{}
	var ok bool
	stat.Transforms_file_size_mb, ok = file_size_mb_zero_if_nonexist(s.transforms_full_path)
	if !ok { return nil, false }
	stat.Stabi_file_size_mb, ok = file_size_mb_zero_if_nonexist(s.stabi_vid_full_path)
	if !ok { return nil, false }
	stat.Is_done = s.is_done
	return stat, true
}


func download_stabi_handler_post(w http.ResponseWriter, r *http.Request) {
// From: https://www.golangprograms.com/example-to-handle-get-and-post-request-in-golang.html
// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
	var err error
    if err = r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
    }
    ID := r.FormValue("ID")
    Sess := ID2Sess[ID]

	if Sess.is_done {
		_, err = Sess.send_stabi_vid_to_client(w, r); if err != nil { return }
    }

    return
}

func download_stabi_handler_get(w http.ResponseWriter, r *http.Request) {
// From: https://yourbasic.org/golang/http-server-example/
	var err error
	// fmt.Println("in download_stabi_handler_get")
	// fmt.Fprintf(w, "Hello, %s!", r.URL.Path[sess_id_start:])
	sess_id := r.URL.Path[sess_id_start:]
	Sess := ID2Sess[sess_id]
	_, err = Sess.send_stabi_vid_to_client(w, r); if err != nil { return }

    return
}

func status_handler(w http.ResponseWriter, r *http.Request) {
// From: https://www.golangprograms.com/example-to-handle-get-and-post-request-in-golang.html
// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
    if err := r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
    }
    ID := r.FormValue("ID")
    Sess := ID2Sess[ID]

// https://www.alexedwards.net/blog/golang-response-snippets
    StatusResp, ok := Sess.output_files_status()
	if !ok { return }
	// I don't know yet how to decide whether processing is done. So for now, OutputStatus.is_done will be simply "false":
    js, err := json.Marshal(StatusResp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	// if Sess.is_done {
	// 	_, err = Sess.send_stabi_vid_to_client(w, r); if err != nil { return }
    // }

	fmt.Println("------------------------------------------------------------------------------------")
	fmt.Println(StatusResp.Transforms_file_size_mb, StatusResp.Stabi_file_size_mb)
	fmt.Println("------------------------------------------------------------------------------------")

    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
    fmt.Printf("status_handler responded with: %q\n", js)
    return
}

func main() {
    fmt.Println("Hello World")
    NewSelectUploadFileHtmlTmpl()
	NewProgressHtmlTmpl()
    setupRoutes()
}


