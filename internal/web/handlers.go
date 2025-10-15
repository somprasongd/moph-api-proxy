package web

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"moph-ic-proxy/internal/authpayload"
	"moph-ic-proxy/internal/cache"
	"moph-ic-proxy/internal/config"
	"moph-ic-proxy/internal/httpclient"
	"moph-ic-proxy/internal/keygen"
)

// Server bundles dependencies for the HTML endpoints.
type Server struct {
	cfg       config.Config
	templates *template.Template
	tokens    *httpclient.TokenManager
	cache     *cache.Client
	keys      *keygen.Manager
	logger    *log.Logger
}

// NewServer constructs the HTML server.
func NewServer(cfg config.Config, tpl *template.Template, tokens *httpclient.TokenManager, cacheClient *cache.Client, keys *keygen.Manager, logger *log.Logger) *Server {
	return &Server{cfg: cfg, templates: tpl, tokens: tokens, cache: cacheClient, keys: keys, logger: logger}
}

// Register attaches handlers to the provided mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", s.home)
	mux.HandleFunc("/change-password", s.changePassword)
	mux.HandleFunc("/api-key", s.apiKey)
}

type apiDesc struct {
	Tag         string
	Name        string
	Description string
	ProxyHost   string
	URL         string
	Doc         string
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title     string
		UseAPIKey bool
		Host      string
		APIs      []apiDesc
	}{
		Title:     "MOPH API Proxy",
		UseAPIKey: s.cfg.UseAPIKey,
		Host:      "http://" + r.Host,
	}

	if r.TLS != nil {
		data.Host = "https://" + r.Host
	} else if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		data.Host = proto + "://" + r.Host
	}

	appendAPI := func(tag, name, desc, proxyHost, urlPath, doc string) {
		if strings.TrimSpace(proxyHost) == "" {
			return
		}
		data.APIs = append(data.APIs, apiDesc{Tag: tag, Name: name, Description: desc, ProxyHost: proxyHost, URL: urlPath, Doc: doc})
	}

	appendAPI("MOPH IC", "MOPH Immunization Center", "API MOPH Immunization Center", s.cfg.MophICAPI, "api/ImmunizationTarget?cid=xxxxxxxxxxxxx", "https://docs.google.com/document/d/1Inyhfrte0pECsD8YoForTL2W8B2hOxezf0GpTGEjJr8/edit")
	appendAPI("MOPH IC", "EPIDEM Center", "API EPIDEM Center", s.cfg.EpidemAPI, "api/SendEPIDEM?endpoint=epidem", "https://ddc.moph.go.th/viralpneumonia/file/g_surveillance/g_api_epidem_0165.pdf")
	appendAPI("MOPH IC", "MOPH-PHR", "ส่งข้อมูลเข้าระบบ MOPH-PHR", s.cfg.MophPhrAPI, "api/RequestTokenv1?endpoint=phr", "https://docs.google.com/document/d/1ZWCBJnxVCtjqmBGNjj1sLnYv11dVzUvnweui-26NDJ0/edit")
	appendAPI("MOPH IC", "MOPH Claim-NHSO", "(DMHT/EPI/dT Services)", s.cfg.MophClaimAPI, "api/v1/opd/service-admissions/dmht?endpoint=claim", "https://docs.google.com/document/d/1iiybB2y7NJkEhXTdS4DbYe3-Fs7aka7MlEns81lzODQ/edit")
	appendAPI("FDH", "Financial Data Hub (FDH)", "ศูนย์กลางข้อมูลด้านการเงิน Financial Data Hub (FDH) กระทรวงสาธารณสุข", s.cfg.FDHAPI, "api/v1/data_hub/16_files?endpoint=fdh", "https://drive.google.com/file/d/17XqRmSEOnXoJVwzmCwteuVdy-Gp_SUyW")
	appendAPI("FDH", "Minimal Data Set", "สำหรับเชื่อมต่อการจองเคลมผ่าน Financial Data Hub", s.cfg.FDHAPI, "api/v1/reservation?endpoint=fdh", "https://docs.google.com/document/d/1yDwflxOG_EG9HkEWbewfk446kmkGQ_2KiJhyqg2iY2E")

	s.render(w, "index", data)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	app := r.URL.Query().Get("app")
	if app == "" {
		app = "mophic"
	}

	data := struct {
		App     string
		Status  string
		Message string
	}{
		App: app,
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.logger.Printf("ERROR parse change-password form: %v", err)
			s.writeError(w, http.StatusBadRequest, "invalid form payload")
			return
		}

		username := strings.TrimSpace(r.PostFormValue("username"))
		password := strings.TrimSpace(r.PostFormValue("password"))

		if username == "" || password == "" {
			data.Status = "error"
			data.Message = "username and password are required"
			s.render(w, "change-password", data)
			return
		}

		_, err := s.tokens.GetToken(ctx, httpclient.GetTokenOptions{Force: true, Username: username, Password: password, App: app})
		if err != nil {
			s.logger.Printf("ERROR change password token retrieval failed: %v", err)
			data.Status = "error"
			data.Message = "Invalid username or password"
		} else {
			data.Status = "success"
			if app == "mophic" {
				data.Message = "Created new MOPH IC Token"
			} else {
				data.Message = "Created new FDH Token"
			}
		}
	}

	s.render(w, "change-password", data)
}

func (s *Server) apiKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	data := struct {
		Status  string
		Message string
	}{}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.logger.Printf("ERROR parse api-key form: %v", err)
			s.writeError(w, http.StatusBadRequest, "invalid form payload")
			return
		}
		username := strings.TrimSpace(r.PostFormValue("username"))
		password := strings.TrimSpace(r.PostFormValue("password"))

		matched, err := authpayload.IsCurrent(ctx, s.cache, s.cfg, "mophic", username, password)
		if err != nil {
			s.logger.Printf("ERROR checking mophic payload: %v", err)
		}
		if !matched {
			matched, err = authpayload.IsCurrent(ctx, s.cache, s.cfg, "fdh", username, password)
			if err != nil {
				s.logger.Printf("ERROR checking fdh payload: %v", err)
			}
		}

		if !matched {
			data.Status = "error"
			data.Message = "Invalid username or password"
		} else {
			data.Status = "success"
			data.Message = s.keys.APIKey()
		}
	}

	s.render(w, "api-key", data)
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	tpl, err := s.templates.Clone()
	if err != nil {
		s.logger.Printf("ERROR clone templates: %v", err)
		s.writeError(w, http.StatusInternalServerError, "template error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Printf("ERROR render template %s: %v", name, err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}
