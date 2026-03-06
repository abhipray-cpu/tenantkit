package httpgin

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHeaderResolver(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		value   string
		wantErr bool
		want    string
	}{
		{"default header", "", "t1", false, "t1"},
		{"custom header", "X-Custom", "t2", false, "t2"},
		{"missing", "", "", true, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest("GET", "/", nil)
			c.Request = req
			
			if tt.value != "" {
				h := tt.header
				if h == "" {
					h = "X-Tenant-ID"
				}
				req.Header.Set(h, tt.value)
			}
			
			r := &HeaderResolver{HeaderName: tt.header}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSubdomainResolver(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		base    string
		wantErr bool
		want    string
	}{
		{"with base", "t1.example.com", "example.com", false, "t1"},
		{"with port", "t2.example.com:8080", "example.com", false, "t2"},
		{"no base", "localhost", "", false, "localhost"},
		{"multi-part no base", "t1.example.com", "", false, "t1"},
		{"no subdomain", "example.com", "example.com", true, ""},
		{"wrong base", "t1.other.com", "example.com", true, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			c.Request = req
			
			r := &SubdomainResolver{BaseDomain: tt.base}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPathResolver(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		index   int
		wantErr bool
		want    string
	}{
		{"index 0", "/t1/users", 0, false, "t1"},
		{"index 1", "/tenants/t2/users", 1, false, "t2"},
		{"out of range", "/api", 5, true, ""},
		{"negative", "/api", -1, true, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest("GET", tt.path, nil)
			c.Request = req
			
			r := &PathResolver{PathIndex: tt.index}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParamResolver(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		value   string
		wantErr bool
		want    string
	}{
		{"default param", "", "t1", false, "t1"},
		{"custom param", "tid", "t2", false, "t2"},
		{"missing", "tenantID", "", true, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest("GET", "/", nil)
			c.Request = req
			
			pname := tt.param
			if pname == "" {
				pname = "tenantID"
			}
			if tt.value != "" {
				c.Params = gin.Params{{Key: pname, Value: tt.value}}
			}
			
			r := &ParamResolver{ParamName: tt.param}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestQueryParamResolver(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		value   string
		wantErr bool
		want    string
	}{
		{"default", "", "t1", false, "t1"},
		{"custom", "tid", "t2", false, "t2"},
		{"missing", "tenant", "", true, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			
			url := "/"
			if tt.value != "" {
				p := tt.param
				if p == "" {
					p = "tenant"
				}
				url += "?" + p + "=" + tt.value
			}
			
			req := httptest.NewRequest("GET", url, nil)
			c.Request = req
			
			r := &QueryParamResolver{ParamName: tt.param}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestChainResolver(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*gin.Context)
		wantErr bool
		want    string
	}{
		{
			"first succeeds",
			func(c *gin.Context) {
				c.Request.Header.Set("X-Tenant-ID", "h-tenant")
			},
			false,
			"h-tenant",
		},
		{
			"fallback",
			func(c *gin.Context) {},
			false,
			"fallback",
		},
		{
			"all fail",
			func(c *gin.Context) {},
			true,
			"",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/", nil)
			tt.setup(c)
			
			var resolvers []TenantResolver
			if tt.name == "all fail" {
				resolvers = []TenantResolver{&HeaderResolver{}, &QueryParamResolver{}}
			} else {
				resolvers = []TenantResolver{&HeaderResolver{}, &StaticResolver{TenantID: "fallback"}}
			}
			
			r := &ChainResolver{Resolvers: resolvers}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestStaticResolver(t *testing.T) {
	tests := []struct {
		name    string
		tenant  string
		wantErr bool
	}{
		{"valid", "static", false},
		{"empty", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/", nil)
			
			r := &StaticResolver{TenantID: tt.tenant}
			got, err := r.Resolve(c)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.tenant {
				t.Errorf("got %s, want %s", got, tt.tenant)
			}
		})
	}
}
