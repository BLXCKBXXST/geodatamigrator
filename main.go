// geo-conv — minimal converter from v2ray/xray geoip.dat and geosite.dat
// to sing-box geoip.db and geosite.db format.
//
// Usage:
//
//	geo-conv geoip  -i geoip.dat  -o geoip.db
//	geo-conv geosite -i geosite.dat -o geosite.db
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	pb "github.com/BLXCKBXXST/geodatamigrator/proto"
	"github.com/BLXCKBXXST/geodatamigrator/geositedb"
	"google.golang.org/protobuf/proto"
)

const version = "1.2.0"

func main() {
	if len(os.Args) < 2 {
		printUsageTo(os.Stderr)
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "geoip":
		runGeoIP(os.Args[2:])
	case "geosite":
		runGeoSite(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("geo-conv v%s\n", version)
	case "help", "-h", "--help":
		printUsageTo(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
		printUsageTo(os.Stderr)
		os.Exit(1)
	}
}

func printUsageTo(w *os.File) {
	fmt.Fprintf(w, `geo-conv v%s — convert v2ray/xray geo databases to sing-box format

Usage:
  geo-conv geoip   -i <input.dat> -o <output.db>
  geo-conv geosite -i <input.dat> -o <output.db>
  geo-conv version
  geo-conv help

Commands:
  geoip     Convert geoip.dat (v2ray/xray) → geoip.db (sing-box MMDB)
  geosite   Convert geosite.dat (v2ray/xray) → geosite.db (sing-box binary)
  version   Show version
  help      Show this help

Options:
  -i <path>   Path to input .dat file (default: geoip.dat / geosite.dat)
  -o <path>   Path to output .db file  (default: geoip.db / geosite.db)
`, version)
}

// parseFlags extracts -i and -o from args. Returns (input, output).
// If -h or --help is found, prints usage and exits.
func parseFlags(args []string, defaultIn, defaultOut string) (string, string) {
	input := defaultIn
	output := defaultOut
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help", "help":
			printUsageTo(os.Stdout)
			os.Exit(0)
		case "-i":
			if i+1 < len(args) {
				input = args[i+1]
				i++
			}
		case "-o":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		}
	}
	return input, output
}

// ---------------------------------------------------------------------------
// GeoIP conversion: v2ray protobuf → sing-box MMDB
// ---------------------------------------------------------------------------

func runGeoIP(args []string) {
	inputPath, outputPath := parseFlags(args, "geoip.dat", "geoip.db")

	if err := checkFileExists(inputPath); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Reading %s ...\n", inputPath)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fatal("cannot read input file: %v", err)
	}

	var geoipList pb.GeoIPList
	if err := proto.Unmarshal(data, &geoipList); err != nil {
		fatal("failed to parse geoip.dat (invalid protobuf): %v", err)
	}

	fmt.Fprintf(os.Stderr, "Parsed %d country entries\n", len(geoipList.Entry))

	// Collect all country codes for MMDB Languages metadata.
	// sing-box uses Languages to enumerate available codes.
	codes := make([]string, 0, len(geoipList.Entry))
	for _, entry := range geoipList.Entry {
		codes = append(codes, strings.ToLower(entry.CountryCode))
	}
	sort.Strings(codes)

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "sing-geoip",
		Languages:               codes,
		IPVersion:               6,
		RecordSize:              24,
		Inserter:                inserter.ReplaceWith,
		DisableIPv4Aliasing:     true,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		fatal("cannot create MMDB writer: %v", err)
	}

	var totalCIDRs int
	for _, entry := range geoipList.Entry {
		code := strings.ToLower(entry.CountryCode)
		for _, cidr := range entry.Cidr {
			ipAddr := net.IP(cidr.Ip)
			if ip4 := ipAddr.To4(); ip4 != nil {
				ipAddr = ip4
			}
			ipNet := &net.IPNet{
				IP:   ipAddr,
				Mask: net.CIDRMask(int(cidr.Prefix), len(ipAddr)*8),
			}
			if err := writer.Insert(ipNet, mmdbtype.String(code)); err != nil {
				fatal("MMDB insert error for %s/%d (%s): %v", ipAddr, cidr.Prefix, code, err)
			}
			totalCIDRs++
		}
	}

	fmt.Fprintf(os.Stderr, "Inserted %d CIDR records\n", totalCIDRs)
	fmt.Fprintf(os.Stderr, "Writing %s ...\n", outputPath)

	outFile, err := os.Create(outputPath)
	if err != nil {
		fatal("cannot create output file: %v", err)
	}

	bufWriter := bufio.NewWriter(outFile)
	if _, err := writer.WriteTo(bufWriter); err != nil {
		outFile.Close()
		os.Remove(outputPath)
		fatal("MMDB write error: %v", err)
	}
	if err := bufWriter.Flush(); err != nil {
		outFile.Close()
		os.Remove(outputPath)
		fatal("flush error: %v", err)
	}

	fi, _ := outFile.Stat()
	fmt.Fprintf(os.Stderr, "Done! %s (%.2f MB)\n", outputPath, float64(fi.Size())/1024/1024)
	outFile.Close()
}

// ---------------------------------------------------------------------------
// GeoSite conversion: v2ray protobuf → sing-box geosite.db
// ---------------------------------------------------------------------------

func runGeoSite(args []string) {
	inputPath, outputPath := parseFlags(args, "geosite.dat", "geosite.db")

	if err := checkFileExists(inputPath); err != nil {
		fatal("%v", err)
	}

	fmt.Fprintf(os.Stderr, "Reading %s ...\n", inputPath)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fatal("cannot read input file: %v", err)
	}

	var geositeList pb.GeoSiteList
	if err := proto.Unmarshal(data, &geositeList); err != nil {
		fatal("failed to parse geosite.dat (invalid protobuf): %v", err)
	}

	fmt.Fprintf(os.Stderr, "Parsed %d geosite entries\n", len(geositeList.Entry))

	domainMap := make(map[string][]geositedb.Item)

	for _, entry := range geositeList.Entry {
		code := strings.ToLower(entry.CountryCode)
		domains := make([]geositedb.Item, 0, len(entry.Domain)*2)
		attributes := make(map[string][]*pb.Domain)

		for _, domain := range entry.Domain {
			// Collect attribute-tagged domains.
			if len(domain.Attribute) > 0 {
				for _, attr := range domain.Attribute {
					attributes[attr.Key] = append(attributes[attr.Key], domain)
				}
			}

			// Convert domain type.
			domains = appendDomainItem(domains, domain)
		}
		domainMap[code] = dedup(domains)

		// Generate sub-entries for attribute tags (e.g., "google@cn").
		for attr, attrDomains := range attributes {
			attrItems := make([]geositedb.Item, 0, len(attrDomains)*2)
			for _, domain := range attrDomains {
				attrItems = appendDomainItem(attrItems, domain)
			}
			domainMap[code+"@"+attr] = dedup(attrItems)
		}
	}

	fmt.Fprintf(os.Stderr, "Total codes (with attributes): %d\n", len(domainMap))
	fmt.Fprintf(os.Stderr, "Writing %s ...\n", outputPath)

	outFile, err := os.Create(outputPath)
	if err != nil {
		fatal("cannot create output file: %v", err)
	}

	bufWriter := bufio.NewWriter(outFile)
	if err := geositedb.Write(bufWriter, domainMap); err != nil {
		outFile.Close()
		os.Remove(outputPath)
		fatal("geosite write error: %v", err)
	}
	if err := bufWriter.Flush(); err != nil {
		outFile.Close()
		os.Remove(outputPath)
		fatal("flush error: %v", err)
	}

	fi, _ := outFile.Stat()
	fmt.Fprintf(os.Stderr, "Done! %s (%.2f MB)\n", outputPath, float64(fi.Size())/1024/1024)
	outFile.Close()
}

// appendDomainItem converts a v2ray Domain proto to sing-box geosite items.
func appendDomainItem(items []geositedb.Item, domain *pb.Domain) []geositedb.Item {
	switch domain.Type {
	case pb.Domain_Plain:
		items = append(items, geositedb.Item{
			Type:  geositedb.RuleTypeDomainKeyword,
			Value: domain.Value,
		})
	case pb.Domain_Regex:
		items = append(items, geositedb.Item{
			Type:  geositedb.RuleTypeDomainRegex,
			Value: domain.Value,
		})
	case pb.Domain_RootDomain:
		// RootDomain → exact domain match (if contains dot) + suffix match.
		if strings.Contains(domain.Value, ".") {
			items = append(items, geositedb.Item{
				Type:  geositedb.RuleTypeDomain,
				Value: domain.Value,
			})
		}
		items = append(items, geositedb.Item{
			Type:  geositedb.RuleTypeDomainSuffix,
			Value: "." + domain.Value,
		})
	case pb.Domain_Full:
		items = append(items, geositedb.Item{
			Type:  geositedb.RuleTypeDomain,
			Value: domain.Value,
		})
	}
	return items
}

// dedup removes duplicate items while preserving order.
func dedup(items []geositedb.Item) []geositedb.Item {
	seen := make(map[geositedb.Item]struct{}, len(items))
	result := make([]geositedb.Item, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// checkFileExists provides a clear error message when the input file is missing.
func checkFileExists(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("cannot access input file: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("input path is a directory, not a file: %s", path)
	}
	return nil
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
