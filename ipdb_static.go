package main

import (
  "fmt"
  "time"
  "os"
//  "log"
  "flag"
  "strings"
  "slices"
  _ "embed"

  "regexp"

  "html"
  "runtime/debug"
  "database/sql"
  _ "github.com/go-sql-driver/mysql"

  . "github.com/ShyLionTjmn/m"
  . "github.com/ShyLionTjmn/dbtools"

  "github.com/rohanthewiz/element"
)

const (
  F_ALLOW_LEAFS uint64 = 1 << iota // allow to create leafs off non-root tag
  F_DENY_SELECT // deny selection as value
  F_DISPLAY // display in popup title, root-> ... -> final_tag chain
  F_IN_LABEL // display in tag label, before tag name, root-> ... -> final_tag chain
)


//go:embed styles.css
var styles string

//go:embed ipdb_static.js
var js string

var opt_d bool= false
var opt_b string

func logError(source string, message string) {
  fmt.Fprintln(os.Stderr, source, message)
}

var g_num_reg *regexp.Regexp

func init() {
  g_num_reg = regexp.MustCompile(`^\d+$`)
}

func get_tag_path(tags M, tag_id string, count int, prev_path []string) ([]string) {
  if count > 256 {
    panic("Tags loop detected")
  }

  if !tags.EvM(tag_id) {
    panic("Unknown tag_id: " + tag_id)
  }

  ret := append(prev_path, tag_id)

  parent_id := tags.Vs(tag_id, "tag_parent_id")
  if parent_id == "0" {
    return ret
  } else {
    return get_tag_path(tags, parent_id, count + 1, ret)
  }
}

func full_tag(tags M, tag_id string, b *element.Builder) {
  //b.Ele("span").R())
  tags_path := make([]string, 0)
  tags_path = append(tags_path, get_tag_path(tags, tag_id, 0, []string{})...)

  popups := make([]string, 0)
  labels := make([]string, 0)
  if len(tags_path) > 0 {
    for i := len(tags_path) - 1; i >= 0; i-- {
      t_id := tags_path[i]
      t_flags := tags.Vu(t_id, "tag_flags")
      if (t_flags & F_DISPLAY) != 0 || i == 0 {
        popups = append(popups, tags.Vs(t_id, "tag_name"))
      }
      if (t_flags & F_IN_LABEL) != 0 || i == 0 {
        labels = append(labels, tags.Vs(t_id, "tag_name"))
      }
    }
  }

  b.Ele("span", "class", "tag", "title", html.EscapeString(strings.Join(popups, ":"))).R(
    b.Text(html.EscapeString(strings.Join(labels, ":"))),
  )
}

func main() {

  var f_opt_d *bool = flag.Bool("d", opt_d, "Debug output")
  var f_opt_b *string = flag.String("b", DSN, "Database DSN")

  flag.Parse()



  opt_d = *f_opt_d
  opt_b = *f_opt_b

  defer func() {
    if r := recover(); r != nil {
      out_text := ""

      switch v := r.(type) {
      case string:
        out_text = "Server message:\n"+v;
      case error:
        out_text = v.Error()
      default:
        out_text = "Unknown error";
      }

      out_text = out_text + "\n\n" + string(debug.Stack())

      fmt.Fprint(os.Stderr, out_text)

      fmt.Fprintln(os.Stderr)

      os.Exit(1)
    }
  }()

  var err error
  var db *sql.DB
  var dbres sql.Result
  _ = dbres

  var query string
  _ = query

  db, err = sql.Open("mysql", opt_b)
  if err != nil { panic(err) }

  defer db.Close()

  var us M
  var tags M
  var vds []M
  var vlans M
  var ics M

  us, err = Return_query_M(db, "SELECT * FROM us", "u_id")
  if err != nil { panic(err) }
  _ = us

  tags, err = Return_query_M(db, "SELECT * FROM tags", "tag_id")
  if err != nil { panic(err) }
  _ = tags

  vlans, err = Return_query_M(db, "SELECT * FROM vlans", "vlan_id")
  if err != nil { panic(err) }
  _ = vlans

  vlans_sorted := make([]string, len(vlans))
  i := 0
  for vlan_id, _ := range vlans {
    vlans_sorted[i] = vlan_id
    i++
  }

  slices.SortFunc(vlans_sorted, func(a, b string) int {
    if vlans.Vu(a, "vlan_number") < vlans.Vu(b, "vlan_number") {
      return -1
    } else if vlans.Vu(a, "vlan_number") > vlans.Vu(b, "vlan_number") {
      return 1
    } else {
      return 0
    }
  })

  vds, err = Return_query_A(db, "SELECT * FROM vds ORDER BY vd_name")
  if err != nil { panic(err) }
  _ = vds

  ics, err = Return_query_M(db, "SELECT * FROM ics", "ic_id")
  if err != nil { panic(err) }
  _ = ics

  b := element.NewBuilder()
	e := b.Ele
	t := b.Text

  now := time.Now().Format(time.RFC1123Z)

	_ = b.WriteString("<!DOCTYPE html>\n")
	e("html").R(
    e("head").R(
      e("title").R(t(`Static IPDB`)),
      e("meta", "charset", "UTF-8"),
      e("meta", "http-equiv", "Cache-control", "content", "no-cache"),
      e("link", "rel", "icon", "href", "data:,"),
      e("style").R(t(styles)),
      e("script", "type", "text/javascript").R(t(js)),
    ),
    e("body").R(
      e("div").R(t("Generated: " + now)),
      e("div").R(
        e("label", "class", "button", "onclick", "expandAll();").R(t("Развернуть все")),
        e("label", "class", "button", "onclick", "collapseAll();").R(t("Свернуть все")),
      ),
      e("H1").R(t(`Сети IPv4`)),
      func() (_ any) {

        // listing v4nets

        query = "SELECT * FROM v4nets ORDER BY v4net_addr"
        nets, nets_err :=  Return_query_A(db, query)
        if nets_err != nil { panic(nets_err) }

        for _, row := range nets {
          net_id := row.Vs("v4net_id")
          net_str_addr := v4long2ip(uint32(row.Vu("v4net_addr"))) + "/" + row.Vs("v4net_mask")

          var netcols []M
          query = "SELECT ic_id FROM n4cs INNER JOIN ics ON n4cs.nc_fk_ic_id = ics.ic_id"
          query += " WHERE nc_fk_v4net_id=? ORDER BY ic_sort"

          netcols, err = Return_query_A(db, query, net_id)
          if err != nil { panic(err) }

          var ipvalues []M
          query = "SELECT i4vs.iv_value, i4vs.iv_fk_ic_id, i4vs.ts, i4vs.fk_u_id, v4ips.v4ip_id"+
              " FROM (i4vs INNER JOIN v4ips ON i4vs.iv_fk_v4ip_id=v4ips.v4ip_id)"+
              " WHERE v4ip_fk_v4net_id=?"
          ipvalues, err = Return_query_A(db, query, net_id)
          if err != nil { panic(err) }

          ipvs := make(M)
          for _, ipv_row := range ipvalues {
            ip_id := ipv_row.Vs("v4ip_id")
            ic_id := ipv_row.Vs("iv_fk_ic_id")

            if !ipvs.EvM(ipv_row.Vs("v4ip_id")) {
              ipvs[ip_id] = make(M)
            }
            ipvs[ip_id].(M)[ic_id] = ipv_row
          }

          var ips []M
          query = "SELECT v4ips.* FROM v4ips INNER JOIN v4nets ON v4ip_fk_v4net_id=v4net_id"+
              " WHERE v4ip_fk_v4net_id=? ORDER BY v4ip_addr"
          ips, err = Return_query_A(db, query, net_id)
          if err != nil { panic(err) }

          e("div", "class", "cont").R(
            e("div", "class", "cont_head", "onclick", "elmToggle('v4net_cont" + net_id + "');").R(
              e("span", "class", "netaddr").R(t(net_str_addr)),
              e("span", "class", "netname").R(t(html.EscapeString(row.Vs("v4net_name")))),
              func() (_ any) {
                if(row.Evu("v4net_fk_vlan_id")) {
                  vlan_id := row.Vs("v4net_fk_vlan_id")
                  vlan := vlans.Vs(vlan_id, "vlan_number")
                  vlan_name := vlans.Vs(vlan_id, "vlan_name")
                  vlan_descr := vlans.Vs(vlan_id, "vlan_descr")
                  e("span", "class", "netvlan", "title", vlan_name + "\n" + vlan_descr).R(t("VLAN: " + vlan))
                }
                return
              } (),
            ),
            e("div", "style", "display: none", "class", "cont_data", "id", "v4net_cont" + net_id).R(
              e("div", "class", "netinfo").R(
                func() (_ any) {
                  raw_tags := strings.Split(row.Vs("v4net_tags"), ",")
                  _tags := make([]string, 0)

                  for _, raw_tag := range raw_tags {
                    trimmed := strings.TrimSpace(raw_tag)
                    if g_num_reg.MatchString(trimmed) {
                      _tags = append(_tags, trimmed)
                    }
                  }
                  if len(_tags) > 0 {
                    e("div", "class", "nettags").R(
                      e("label").R(t("Теги: ")),
                      func() (_ any) {
                        for _, tag_id := range _tags {
                          full_tag(tags, tag_id, b)
                        }
                        return
                      } (),
                    )
                  }
                  return
                } (),
                e("div", "class", "netdescrdiv").R(
                  e("textarea", "class", "netdescr", "readonly", "true").R(t(html.EscapeString(row.Vs("v4net_descr")))),
                ),
              ),
              e("div", "class", "netips").R(
                e("table", "class", "ipstable").R(
                  e("thead").R(
                    e("tr").R(
                      e("th"),
                      func () (_ any) {
                        for _, col_row := range netcols {
                          ic_id := col_row.Vs("ic_id")
                          e("th", "title", ics.Vs(ic_id, "ic_type")).R(t(html.EscapeString(ics.Vs(ic_id, "ic_name"))))
                        }
                        return
                      } (),
                    ),
                  ),
                  e("tbody").R(
                    func() (_ any) {
                      for _, iprow := range ips {
                        ip_id := iprow.Vs("v4ip_id")
                        ip_addr := v4long2ip(uint32(iprow.Vu("v4ip_addr")))
                        e("tr").R(
                          e("td").R(t(ip_addr)),
                          func () (_ any) {
                            for _, col_row := range netcols {
                              ic_id := col_row.Vs("ic_id")
                              if ipvs.Evs(ip_id, ic_id, "iv_value") {
                                ip_value := ipvs.Vs(ip_id, ic_id, "iv_value")
                                ic_type := ics.Vs(ic_id, "ic_type")
                                if ic_type == "textarea" {
                                  e("td").R(
                                    e("textarea", "class", "multiline", "readonly", "1").R(t(html.EscapeString(ip_value))),
                                  )
                                } else if ic_type == "tag" || ic_type == "multitag" {
                                  raw_tags := strings.Split(ip_value, ",")
                                  _tags := make([]string, 0)

                                  for _, raw_tag := range raw_tags {
                                    trimmed := strings.TrimSpace(raw_tag)
                                    if g_num_reg.MatchString(trimmed) {
                                      _tags = append(_tags, trimmed)
                                    }
                                  }
                                  e("td").R(
                                    func() (_ any) {
                                      for _, tag_id := range _tags {
                                        full_tag(tags, tag_id, b)
                                      }
                                      return
                                    } (),
                                  )
                                } else {
                                  e("td").R(t(html.EscapeString(ip_value)))
                                }
                              } else {
                                e("td")
                              }
                            }
                            return
                          } (),
                        )
                      }
                      return
                    } (),
                  ),
                ),
              ),
            ),
          )
        }
        return
      } (),
      e("H1").R(t(`VLAN`)),
      func() (_ any) {

        // listing vlan domains

        for _, vd := range vds {
          vd_id := vd.Vs("vd_id")
          e("div", "class", "cont").R(
            e("div", "class", "cont_head", "onclick", "elmToggle('vlans_cont" + vd_id + "');").R(
              e("span", "class", "vdom").R(t(html.EscapeString(vd.Vs("vd_name")))),
            ),
            e("div", "style", "display: none", "class", "cont_data", "id", "vlans_cont" + vd_id).R(
              e("table", "class", "vlanstable").R(
                e("thead").R(
                  e("tr").R(
                    e("th").R(t("Vlan")),
                    e("th").R(t("Имя")),
                    e("th").R(t("Описание")),
                  ),
                ),
                e("tbody").R(
                  func() (_ any) {
                    for _, vlan_id := range vlans_sorted {
                      if vlans.Vs(vlan_id, "vlan_fk_vd_id") == vd_id {
                        e("tr").R(
                          e("td").R(t(vlans.Vs(vlan_id, "vlan_number"))),
                          e("td").R(t(html.EscapeString(vlans.Vs(vlan_id, "vlan_name")))),
                          e("td").R(t(html.EscapeString(vlans.Vs(vlan_id, "vlan_descr")))),
                        )
                      }
                    }
                    return
                  } (),
                ),
              ),
            ),
          )
        }
        return
      } (),
    ),
  )

  fmt.Print(b.String())

  fmt.Println()
}
