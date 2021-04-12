package libv2ray

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"AndroidLibV2rayLite/CoreI"
	"AndroidLibV2rayLite/Process/Escort"
	"AndroidLibV2rayLite/VPN"
	"AndroidLibV2rayLite/shippedBinarys"
	mobasset "golang.org/x/mobile/asset"

	"github.com/2dust/AndroidLibXrayLite/VPN"
	mobasset "golang.org/x/mobile/asset"

	v2core "github.com/xtls/xray-core/core"
	v2net "github.com/xtls/xray-core/common/net"
	v2filesystem "github.com/xtls/xray-core/common/platform/filesystem"
	v2stats "github.com/xtls/xray-core/features/stats"
	v2serial "github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
	v2internet "github.com/xtls/xray-core/transport/internet"

	v2applog "github.com/xtls/xray-core/app/log"
	v2commlog "github.com/xtls/xray-core/common/log"
)

const (
	v2Assert    = "v2ray.location.asset"
	assetperfix = "/dev/libv2rayfs0/asset"
)

/*V2RayPoint V2Ray Point Server
This is territory of Go, so no getter and setters!
*/
type V2RayPoint struct {
	SupportSet   V2RayVPNServiceSupportsSet
	statsManager v2stats.Manager

	dialer    *VPN.ProtectedDialer
	status    *CoreI.Status
	escorter  *Escort.Escorting
	v2rayOP   *sync.Mutex
	closeChan chan struct{}

	PackageName          string
	DomainName           string
	ConfigureFileContent string
	EnableLocalDNS       bool
	ForwardIpv6          bool
}

/*V2RayVPNServiceSupportsSet To support Android VPN mode*/
type V2RayVPNServiceSupportsSet interface {
	Setup(Conf string) int
	Prepare() int
	Shutdown() int
	Protect(int) int
	OnEmitStatus(int, string) int
	SendFd() int
}

/*RunLoop Run V2Ray main loop
 */
func (v *V2RayPoint) RunLoop() (err error) {
	v.v2rayOP.Lock()
	defer v.v2rayOP.Unlock()
	//Construct Context
	v.status.PackageName = v.PackageName

	if !v.status.IsRunning {
		v.closeChan = make(chan struct{})
		v.dialer.PrepareResolveChan()
		go v.dialer.PrepareDomain(v.DomainName, v.closeChan)
		go func() {
			select {
			// wait until resolved
			case <-v.dialer.ResolveChan():
				// shutdown VPNService if server name can not reolved
				if !v.dialer.IsVServerReady() {
					log.Println("vServer cannot resolved, shutdown")
					v.StopLoop()
					v.SupportSet.Shutdown()
				}

			// stop waiting if manually closed
			case <-v.closeChan:
			}
		}()

		err = v.pointloop()
	}
	return
}

/*StopLoop Stop V2Ray main loop
 */
func (v *V2RayPoint) StopLoop() (err error) {
	v.v2rayOP.Lock()
	defer v.v2rayOP.Unlock()
	if v.status.IsRunning {
		close(v.closeChan)
		v.shutdownInit()
		v.SupportSet.OnEmitStatus(0, "Closed")
	}
	return
}

//Delegate Funcation
func (v *V2RayPoint) GetIsRunning() bool {
	return v.status.IsRunning
}

//Delegate Funcation
func (v V2RayPoint) QueryStats(tag string, direct string) int64 {
	if v.statsManager == nil {
		return 0
	}
	counter := v.statsManager.GetCounter(fmt.Sprintf("inbound>>>%s>>>traffic>>>%s", tag, direct))
	if counter == nil {
		return 0
	}
	return counter.Set(0)
}

func (v *V2RayPoint) shutdownInit() {
	v.status.IsRunning = false
	v.status.Vpoint.Close()
	v.status.Vpoint = nil
	v.statsManager = nil
	v.escorter.EscortingDown()
}

func (v *V2RayPoint) pointloop() error {
	if err := v.runTun2socks(); err != nil {
		log.Println(err)
		return err
	}

	log.Printf("EnableLocalDNS: %v\nForwardIpv6: %v\nDomainName: %s",
		v.EnableLocalDNS,
		v.ForwardIpv6,
		v.DomainName)

	log.Println("loading v2ray config")
	config, err := v2serial.LoadJSONConfig(strings.NewReader(v.ConfigureFileContent))
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println("new v2ray core")
	inst, err := v2core.New(config)
	if err != nil {
		log.Println(err)
		return err
	}
	v.status.Vpoint = inst
	v.statsManager = inst.GetFeature(v2stats.ManagerType()).(v2stats.Manager)

	log.Println("start v2ray core")
	v.status.IsRunning = true
	if err := v.status.Vpoint.Start(); err != nil {
		v.status.IsRunning = false
		log.Println(err)
		return err
	}

	v.SupportSet.Prepare()
	v.SupportSet.Setup(v.status.GetVPNSetupArg(v.EnableLocalDNS, v.ForwardIpv6))
	v.SupportSet.OnEmitStatus(0, "Running")
	return nil
}

func initV2Env() {
	if os.Getenv(v2Assert) != "" {
		return
	}
	//Initialize asset API, Since Raymond Will not let notify the asset location inside Process,
	//We need to set location outside V2Ray
	os.Setenv(v2Assert, assetperfix)
	//Now we handle read
	v2filesystem.NewFileReader = func(path string) (io.ReadCloser, error) {
		if strings.HasPrefix(path, assetperfix) {
			p := path[len(assetperfix)+1:]
			//is it overridden?
			//by, ok := overridedAssets[p]
			//if ok {
			//	return os.Open(by)
			//}
			return mobasset.Open(p)
		}
		return os.Open(path)
	}
}

//Delegate Funcation
func TestConfig(ConfigureFileContent string) error {
	initV2Env()
	_, err := v2serial.LoadJSONConfig(strings.NewReader(ConfigureFileContent))
	return err
}

/*NewV2RayPoint new V2RayPoint*/
func NewV2RayPoint(s V2RayVPNServiceSupportsSet) *V2RayPoint {
	initV2Env()

	// inject our own log writer
	v2applog.RegisterHandlerCreator(v2applog.LogType_Console,
		func(lt v2applog.LogType,
			options v2applog.HandlerCreatorOptions) (v2commlog.Handler, error) {
			return v2commlog.NewLogger(createStdoutLogWriter()), nil
		})

	dialer := VPN.NewPreotectedDialer(s)
	v2internet.UseAlternativeSystemDialer(dialer)
	status := &CoreI.Status{}
	return &V2RayPoint{
		SupportSet: s,
		v2rayOP:    new(sync.Mutex),
		status:     status,
		dialer:     dialer,
		escorter:   &Escort.Escorting{Status: status},
	}
}

func (v V2RayPoint) runTun2socks() error {
	shipb := shippedBinarys.FirstRun{Status: v.status}
	if err := shipb.CheckAndExport(); err != nil {
		log.Println(err)
		return err
	}

	v.escorter.EscortingUp()
	go v.escorter.EscortRun(
		v.status.GetApp("tun2socks"),
		v.status.GetTun2socksArgs(v.EnableLocalDNS, v.ForwardIpv6), "",
		v.SupportSet.SendFd)

	return nil
}

/*CheckVersion int
This func will return libv2ray binding version.
*/
func CheckVersion() int {
	return CoreI.CheckVersion()
}

/*CheckVersionX string
This func will return libv2ray binding version and V2Ray version used.
*/
func CheckVersionX() string {
	return fmt.Sprintf("Libv2rayLite V%d, Core V%s", CheckVersion(), v2core.Version())
}

func measureInstDelay(ctx context.Context, inst *v2core.Instance) (int64, error) {
	if inst == nil {
		return -1, errors.New("core instance nil")
	}

	tr := &http.Transport{
		TLSHandshakeTimeout: 6 * time.Second,
		DisableKeepAlives:   true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := v2net.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
			if err != nil {
				return nil, err
			}
			return v2core.Dial(ctx, inst, dest)
		},
	}

	c := &http.Client{
		Transport: tr,
		Timeout:   12 * time.Second,
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://www.google.com/generate_204", nil)
	start := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		return -1, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return -1, fmt.Errorf("status != 204: %s", resp.Status)
	}
	resp.Body.Close()
	return time.Since(start).Milliseconds(), nil
}

