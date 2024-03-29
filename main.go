package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var (
	domain          = ""
	rr              = ""
	regionId        = ""
	accessKeyId     = ""
	accessKeySecret = ""
)

func genRandIpaddr() string {
	rand.Seed(time.Now().Unix())
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	return ip
}

func getExternIp() (string, error) {
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://myexternalip.com/raw")
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return string(b), nil
	}
	return "", errors.New("try too many times")
}

func isSubDomainExist(client *alidns.Client, subDomain string) (recordId string, err error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.Type = "A"
	request.SubDomain = subDomain
	response, err := client.DescribeSubDomainRecords(request)
	if err != nil {
		return "", err
	}

	if response.TotalCount == 0 {
		return "", nil
	}

	return response.DomainRecords.Record[0].RecordId, nil
}

func getSubDomainIp(client *alidns.Client, subDomain string) (ip string, err error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.Type = "A"
	request.SubDomain = subDomain
	response, err := client.DescribeSubDomainRecords(request)
	if err != nil {
		return "", err
	}

	if response.TotalCount == 0 {
		return "", errors.New("can't find submodule")
	}

	return response.DomainRecords.Record[0].Value, nil
}

func addSubDomain(client *alidns.Client, rr, domain, ipValue string) (recordId string, err error) {
	request := alidns.CreateAddDomainRecordRequest()
	request.DomainName = domain
	request.Type = "A"
	request.TTL = "600"
	request.Value = ipValue
	request.RR = rr

	response, err := client.AddDomainRecord(request)
	if err != nil {
		return "", err
	}
	return response.RecordId, nil
}

func updateSubDomain(client *alidns.Client, recordId, rr, ipValue string) error {
	request := alidns.CreateUpdateDomainRecordRequest()
	request.RecordId = recordId
	request.Value = ipValue
	request.RR = rr
	request.Type = "A"

	response, err := client.UpdateDomainRecord(request)
	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		return errors.New(response.String())
	}
	return nil

}

func init() {
	flag.StringVar(&domain, "domain", "", "the domain")
	flag.StringVar(&rr, "rr", "", "subdomain prefix")
	flag.StringVar(&regionId, "regionId", "cn-hangzhou", "regionId of aliyun")
	flag.StringVar(&accessKeyId, "accessKeyId", "", "accessKeyId  of aliyun")
	flag.StringVar(&accessKeySecret, "accessKeySecret", "", "accessKeySecret of aliyun")
	flag.Parse()

	if domain == "" || rr == "" || regionId == "" || accessKeyId == "" || accessKeySecret == "" {
		flag.Usage()
		log.Fatalln("invalid argument")
	}
}

func main() {
	var ip string
	log.Println(ip)
	client, err := alidns.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)
	if err != nil {
		log.Fatalln(err)
	}

	//查询域名是否存在
	recordId, err := isSubDomainExist(client, rr+"."+domain)
	if err != nil {
		log.Fatalln(err)
	}
	if recordId == "" {
		ip, err = getExternIp()
		if err != nil {
			log.Fatalln(err)
		}
		recordId, err = addSubDomain(client, rr, domain, ip)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("subDomain is not existed,created subDomain named %s\n", rr+"."+domain)
	} else {
		ip, err = getSubDomainIp(client, rr+"."+domain)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("subDomain is existed")
	}

	for {
		newIp, err := getExternIp()
		for ; err != nil; newIp, err = getExternIp() {
			log.Println("geting external ip falid,try once a more")
		}

		if newIp != ip {
			err = updateSubDomain(client, recordId, rr, newIp)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("update IP from %s to %s\n", ip, newIp)
			ip = newIp
		} else {
			log.Printf("IP is not changed,current IP is %s\n", ip)
		}

		time.Sleep(time.Minute)
	}

}
