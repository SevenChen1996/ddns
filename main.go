package main

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const subDomain = ""
const domain = ""
const rr = ""


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

func main() {
	Ip, err := getExternIp()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(Ip)
	client, err := alidns.NewClientWithAccessKey()
	if err != nil {
		log.Fatalln(err)
	}

	//查询域名是否存在
	recordId, err := isSubDomainExist(client, subDomain)
	if err != nil {
		log.Fatalln(err)
	}
	if recordId == "" {
		recordId, err = addSubDomain(client, rr, domain, Ip)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		err = updateSubDomain(client, recordId, rr, Ip)
		if err != nil {
			log.Fatalln(err)
		}
	}

	for {
		newIp, err := getExternIp()
		for ; err != nil; newIp, err = getExternIp() {
		}

		if newIp != Ip {
			err = updateSubDomain(client, recordId, rr, Ip)
			if err != nil {
				continue
			}
			Ip = newIp
		}

		time.Sleep(time.Minute)
	}

}
