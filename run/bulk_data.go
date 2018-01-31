package run

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/AlphaHat/gcp-alpha-hat/cache"
)

var bogusData = `
{"EntityData":[{"Data":[{"Data":[{"Time":"2016-10-01T00:00:00Z","Data":191.34989926124916},{"Time":"2016-10-02T00:00:00Z","Data":188.42985074626867},{"Time":"2016-10-03T00:00:00Z","Data":177.33894464227907},{"Time":"2016-10-04T00:00:00Z","Data":171.44414295322247},{"Time":"2016-10-05T00:00:00Z","Data":168.05024182487549},{"Time":"2016-10-06T00:00:00Z","Data":168.3968965140422},{"Time":"2016-10-07T00:00:00Z","Data":167.48283987296384},{"Time":"2016-10-08T00:00:00Z","Data":186.91946392186497},{"Time":"2016-10-09T00:00:00Z","Data":182.74590731210128},{"Time":"2016-10-10T00:00:00Z","Data":181.00424182824264},{"Time":"2016-10-11T00:00:00Z","Data":170.8267037920976},{"Time":"2016-10-12T00:00:00Z","Data":155.51114295410838},{"Time":"2016-10-13T00:00:00Z","Data":162.6151691604629},{"Time":"2016-10-14T00:00:00Z","Data":161.2709064364931},{"Time":"2016-10-15T00:00:00Z","Data":178.26647058152923},{"Time":"2016-10-16T00:00:00Z","Data":177.30875547885583},{"Time":"2016-10-17T00:00:00Z","Data":161.72091590512372},{"Time":"2016-10-18T00:00:00Z","Data":140.79589216944802},{"Time":"2016-10-19T00:00:00Z","Data":146.2049861495845},{"Time":"2016-10-20T00:00:00Z","Data":152.16630415760395},{"Time":"2016-10-21T00:00:00Z","Data":155.00929211015375},{"Time":"2016-10-22T00:00:00Z","Data":169.8567335243553},{"Time":"2016-10-23T00:00:00Z","Data":167.86244281737126},{"Time":"2016-10-24T00:00:00Z","Data":158.9726963900596},{"Time":"2016-10-25T00:00:00Z","Data":153.26239710901427},{"Time":"2016-10-26T00:00:00Z","Data":156.06694560669456},{"Time":"2016-10-27T00:00:00Z","Data":157.77675300386852},{"Time":"2016-10-28T00:00:00Z","Data":155.20628683693516},{"Time":"2016-10-29T00:00:00Z","Data":174.62302038342048},{"Time":"2016-10-30T00:00:00Z","Data":178.29429386075722},{"Time":"2016-10-31T00:00:00Z","Data":146.7188640435273},{"Time":"2016-11-01T00:00:00Z","Data":166.9577916499628},{"Time":"2016-11-02T00:00:00Z","Data":165.79568112806746},{"Time":"2016-11-03T00:00:00Z","Data":171.89617563922195},{"Time":"2016-11-04T00:00:00Z","Data":174.1215398079436},{"Time":"2016-11-05T00:00:00Z","Data":198.63677873073556},{"Time":"2016-11-06T00:00:00Z","Data":186.85233565419568},{"Time":"2016-11-07T00:00:00Z","Data":170.27747127055468},{"Time":"2016-11-08T00:00:00Z","Data":161.93230141180473},{"Time":"2016-11-09T00:00:00Z","Data":164.13155053124126},{"Time":"2016-11-10T00:00:00Z","Data":170.15856070072428},{"Time":"2016-11-11T00:00:00Z","Data":166.17619493908154},{"Time":"2016-11-12T00:00:00Z","Data":197.36156259180984},{"Time":"2016-11-13T00:00:00Z","Data":198.35516653938788},{"Time":"2016-11-14T00:00:00Z","Data":176.93251187774035},{"Time":"2016-11-15T00:00:00Z","Data":173.84140061791967},{"Time":"2016-11-16T00:00:00Z","Data":171.9402454657376},{"Time":"2016-11-17T00:00:00Z","Data":174.36303791295867},{"Time":"2016-11-18T00:00:00Z","Data":178.92085066494815},{"Time":"2016-11-19T00:00:00Z","Data":216.777958789094},{"Time":"2016-11-20T00:00:00Z","Data":223.7086244416447},{"Time":"2016-11-21T00:00:00Z","Data":189.49254538692992},{"Time":"2016-11-22T00:00:00Z","Data":186.87934253112263},{"Time":"2016-11-23T00:00:00Z","Data":210.6645926797113},{"Time":"2016-11-24T00:00:00Z","Data":621.2080605434103},{"Time":"2016-11-25T00:00:00Z","Data":452.38553131164383},{"Time":"2016-11-26T00:00:00Z","Data":308.49220103986136},{"Time":"2016-11-27T00:00:00Z","Data":266.82200012263166},{"Time":"2016-11-28T00:00:00Z","Data":259.50780487257464},{"Time":"2016-11-29T00:00:00Z","Data":203.07088287882766},{"Time":"2016-11-30T00:00:00Z","Data":192.06231360656355},{"Time":"2016-12-01T00:00:00Z","Data":184.64860196480623},{"Time":"2016-12-02T00:00:00Z","Data":186.01557881559273},{"Time":"2016-12-03T00:00:00Z","Data":219.9332386680445},{"Time":"2016-12-04T00:00:00Z","Data":216.52199254772165},{"Time":"2016-12-05T00:00:00Z","Data":192.23598715926266},{"Time":"2016-12-06T00:00:00Z","Data":194.3613700364557},{"Time":"2016-12-07T00:00:00Z","Data":189.02668169860954},{"Time":"2016-12-08T00:00:00Z","Data":195.87183487339493},{"Time":"2016-12-09T00:00:00Z","Data":206.37835225899974},{"Time":"2016-12-10T00:00:00Z","Data":237.87512595212763},{"Time":"2016-12-11T00:00:00Z","Data":230.42178936814335},{"Time":"2016-12-12T00:00:00Z","Data":219.71913095076982},{"Time":"2016-12-13T00:00:00Z","Data":204.25561967795312},{"Time":"2016-12-14T00:00:00Z","Data":207.83479926586364},{"Time":"2016-12-15T00:00:00Z","Data":212.37690692389563},{"Time":"2016-12-16T00:00:00Z","Data":226.48374771161397},{"Time":"2016-12-17T00:00:00Z","Data":266.89770004467243},{"Time":"2016-12-18T00:00:00Z","Data":267.7424981105122},{"Time":"2016-12-19T00:00:00Z","Data":255.6156235108533},{"Time":"2016-12-20T00:00:00Z","Data":261.28103246473233},{"Time":"2016-12-21T00:00:00Z","Data":248.66118971550281},{"Time":"2016-12-22T00:00:00Z","Data":270.75329257171904},{"Time":"2016-12-23T00:00:00Z","Data":290.6159117740217},{"Time":"2016-12-24T00:00:00Z","Data":284.40650601995776},{"Time":"2016-12-25T00:00:00Z","Data":30.85837488209136},{"Time":"2016-12-26T00:00:00Z","Data":311.1771738779203},{"Time":"2016-12-27T00:00:00Z","Data":233.59390284358176},{"Time":"2016-12-28T00:00:00Z","Data":212.50640052461844},{"Time":"2016-12-29T00:00:00Z","Data":210.72782897026764},{"Time":"2016-12-30T00:00:00Z","Data":203.6931396375628},{"Time":"2016-12-31T00:00:00Z","Data":194.39269285938042},{"Time":"2017-01-01T00:00:00Z","Data":224.33067550496474},{"Time":"2017-01-02T00:00:00Z","Data":218.77536061153432},{"Time":"2017-01-03T00:00:00Z","Data":180.9542958126541},{"Time":"2017-01-04T00:00:00Z","Data":178.2562182401712},{"Time":"2017-01-05T00:00:00Z","Data":171.43040988353388},{"Time":"2017-01-06T00:00:00Z","Data":159.23653288251623},{"Time":"2017-01-07T00:00:00Z","Data":188.940213804936},{"Time":"2017-01-08T00:00:00Z","Data":183.92624728850328},{"Time":"2017-01-09T00:00:00Z","Data":177.2319033930708},{"Time":"2017-01-10T00:00:00Z","Data":181.48992112182296},{"Time":"2017-01-11T00:00:00Z","Data":165.02655130157078},{"Time":"2017-01-12T00:00:00Z","Data":175.21346905301752},{"Time":"2017-01-13T00:00:00Z","Data":171.77574996720944},{"Time":"2017-01-14T00:00:00Z","Data":201.10410094637223},{"Time":"2017-01-15T00:00:00Z","Data":193.6098790228843},{"Time":"2017-01-16T00:00:00Z","Data":188.72019409378674},{"Time":"2017-01-17T00:00:00Z","Data":175.35048912704843},{"Time":"2017-01-18T00:00:00Z","Data":174.15315765669263},{"Time":"2017-01-19T00:00:00Z","Data":171.4510796261593},{"Time":"2017-01-20T00:00:00Z","Data":168.3362479404005},{"Time":"2017-01-21T00:00:00Z","Data":200.77270859021783},{"Time":"2017-01-22T00:00:00Z","Data":193.18152939083586},{"Time":"2017-01-23T00:00:00Z","Data":174.0656041059302},{"Time":"2017-01-24T00:00:00Z","Data":164.25408945093983},{"Time":"2017-01-25T00:00:00Z","Data":159.74982166513806},{"Time":"2017-01-26T00:00:00Z","Data":161.41396933560475},{"Time":"2017-01-27T00:00:00Z","Data":162.83703711472108},{"Time":"2017-01-28T00:00:00Z","Data":194.4199657482999},{"Time":"2017-01-29T00:00:00Z","Data":182.91815737428058},{"Time":"2017-01-30T00:00:00Z","Data":171.66711919630737},{"Time":"2017-01-31T00:00:00Z","Data":163.9893110544583},{"Time":"2017-02-01T00:00:00Z","Data":162.00706812694634},{"Time":"2017-02-02T00:00:00Z","Data":165.07116991144096},{"Time":"2017-02-03T00:00:00Z","Data":165.4745969272571},{"Time":"2017-02-04T00:00:00Z","Data":184.90548076694247},{"Time":"2017-02-05T00:00:00Z","Data":177.8545669592644},{"Time":"2017-02-06T00:00:00Z","Data":170.18695888577523},{"Time":"2017-02-07T00:00:00Z","Data":154.46320054017556},{"Time":"2017-02-08T00:00:00Z","Data":155.30349781540818},{"Time":"2017-02-09T00:00:00Z","Data":170.7413571993779},{"Time":"2017-02-10T00:00:00Z","Data":168.36505991692317},{"Time":"2017-02-11T00:00:00Z","Data":189.1493250392792},{"Time":"2017-02-12T00:00:00Z","Data":189.83668437646597},{"Time":"2017-02-13T00:00:00Z","Data":169.91020707348358},{"Time":"2017-02-14T00:00:00Z","Data":149.00442714459865},{"Time":"2017-02-15T00:00:00Z","Data":170.3088633075423},{"Time":"2017-02-16T00:00:00Z","Data":174.79061160888196},{"Time":"2017-02-17T00:00:00Z","Data":173.43561079066816},{"Time":"2017-02-18T00:00:00Z","Data":195.1378337312785},{"Time":"2017-02-19T00:00:00Z","Data":188.09107184399244},{"Time":"2017-02-20T00:00:00Z","Data":194.30528462493348},{"Time":"2017-02-21T00:00:00Z","Data":167.08767095325578},{"Time":"2017-02-22T00:00:00Z","Data":167.90420928402833},{"Time":"2017-02-23T00:00:00Z","Data":177.8528178854215},{"Time":"2017-02-24T00:00:00Z","Data":167.27321355226331},{"Time":"2017-02-25T00:00:00Z","Data":191.27003386897664},{"Time":"2017-02-26T00:00:00Z","Data":190.82727411399807},{"Time":"2017-02-27T00:00:00Z","Data":169.43751550848233},{"Time":"2017-02-28T00:00:00Z","Data":173.3762554104725},{"Time":"2017-03-01T00:00:00Z","Data":168.4174292869945},{"Time":"2017-03-02T00:00:00Z","Data":183.09028644002447},{"Time":"2017-03-03T00:00:00Z","Data":180.77226485262628},{"Time":"2017-03-04T00:00:00Z","Data":202.77584786156308},{"Time":"2017-03-05T00:00:00Z","Data":201.62152622782338},{"Time":"2017-03-06T00:00:00Z","Data":174.82639756207223},{"Time":"2017-03-07T00:00:00Z","Data":172.17150476971995},{"Time":"2017-03-08T00:00:00Z","Data":159.60041097755717},{"Time":"2017-03-09T00:00:00Z","Data":162.81548281314048},{"Time":"2017-03-10T00:00:00Z","Data":168.3936222808174},{"Time":"2017-03-11T00:00:00Z","Data":192.72969333414946},{"Time":"2017-03-12T00:00:00Z","Data":186.78910778669746},{"Time":"2017-03-13T00:00:00Z","Data":169.08189185919923},{"Time":"2017-03-14T00:00:00Z","Data":168.60472896120754},{"Time":"2017-03-15T00:00:00Z","Data":161.66790257258572},{"Time":"2017-03-16T00:00:00Z","Data":167.9614156965828},{"Time":"2017-03-17T00:00:00Z","Data":162.2265829677499},{"Time":"2017-03-18T00:00:00Z","Data":187.24115836029193},{"Time":"2017-03-19T00:00:00Z","Data":180.48288345318048},{"Time":"2017-03-20T00:00:00Z","Data":168.6468939807007},{"Time":"2017-03-21T00:00:00Z","Data":168.55979154544764},{"Time":"2017-03-22T00:00:00Z","Data":157.32651023464803},{"Time":"2017-03-23T00:00:00Z","Data":155.60499320232358},{"Time":"2017-03-24T00:00:00Z","Data":159.0006729475101},{"Time":"2017-03-25T00:00:00Z","Data":182.1959972198976},{"Time":"2017-03-26T00:00:00Z","Data":180.28081542218825},{"Time":"2017-03-27T00:00:00Z","Data":162.27748803930865},{"Time":"2017-03-28T00:00:00Z","Data":164.0865214643083},{"Time":"2017-03-29T00:00:00Z","Data":157.03312184437564},{"Time":"2017-03-30T00:00:00Z","Data":163.54927238295846},{"Time":"2017-03-31T00:00:00Z","Data":166.1308840413318},{"Time":"2017-04-01T00:00:00Z","Data":185.29990894162145},{"Time":"2017-04-02T00:00:00Z","Data":173.16743677775196},{"Time":"2017-04-03T00:00:00Z","Data":161.1298879096432},{"Time":"2017-04-04T00:00:00Z","Data":166.5943003590075},{"Time":"2017-04-05T00:00:00Z","Data":157.19375818326213},{"Time":"2017-04-06T00:00:00Z","Data":158.04066543438077},{"Time":"2017-04-07T00:00:00Z","Data":159.3828142759718},{"Time":"2017-04-08T00:00:00Z","Data":187.7985198663331},{"Time":"2017-04-09T00:00:00Z","Data":181.21160002901044},{"Time":"2017-04-10T00:00:00Z","Data":162.455405175831},{"Time":"2017-04-11T00:00:00Z","Data":153.92742063282446},{"Time":"2017-04-12T00:00:00Z","Data":149.35875043664853},{"Time":"2017-04-13T00:00:00Z","Data":147.0074568288854},{"Time":"2017-04-14T00:00:00Z","Data":158.08756182450998},{"Time":"2017-04-15T00:00:00Z","Data":173.25258999190996},{"Time":"2017-04-16T00:00:00Z","Data":21.31277842473983},{"Time":"2017-04-17T00:00:00Z","Data":176.23119831571637},{"Time":"2017-04-18T00:00:00Z","Data":155.69787985865725},{"Time":"2017-04-19T00:00:00Z","Data":151.80815266005027},{"Time":"2017-04-20T00:00:00Z","Data":157.7554553373543},{"Time":"2017-04-21T00:00:00Z","Data":177.09848204016842},{"Time":"2017-04-22T00:00:00Z","Data":182.6903005055318},{"Time":"2017-04-23T00:00:00Z","Data":171.75803555214785},{"Time":"2017-04-24T00:00:00Z","Data":168.91492773016046},{"Time":"2017-04-25T00:00:00Z","Data":159.3273443810736},{"Time":"2017-04-26T00:00:00Z","Data":154.8749277635598},{"Time":"2017-04-27T00:00:00Z","Data":150.3620759018372},{"Time":"2017-04-28T00:00:00Z","Data":154.22861415655922},{"Time":"2017-04-29T00:00:00Z","Data":175.18453618543504},{"Time":"2017-04-30T00:00:00Z","Data":173.36585565826962},{"Time":"2017-05-01T00:00:00Z","Data":156.78881735920407},{"Time":"2017-05-02T00:00:00Z","Data":144.96767217600458},{"Time":"2017-05-03T00:00:00Z","Data":149.37193391556855},{"Time":"2017-05-04T00:00:00Z","Data":154.51933124346917},{"Time":"2017-05-05T00:00:00Z","Data":152.13239190171652},{"Time":"2017-05-06T00:00:00Z","Data":166.45719999633053},{"Time":"2017-05-07T00:00:00Z","Data":170.86406497992718},{"Time":"2017-05-08T00:00:00Z","Data":154.16882008629705},{"Time":"2017-05-09T00:00:00Z","Data":146.35685108749118},{"Time":"2017-05-10T00:00:00Z","Data":141.3074809277657},{"Time":"2017-05-11T00:00:00Z","Data":151.61878932306988},{"Time":"2017-05-12T00:00:00Z","Data":151.21182955639165},{"Time":"2017-05-13T00:00:00Z","Data":178.65152000554147},{"Time":"2017-05-14T00:00:00Z","Data":148.5494807312677},{"Time":"2017-05-15T00:00:00Z","Data":147.18590614490302},{"Time":"2017-05-16T00:00:00Z","Data":146.36156390140528},{"Time":"2017-05-17T00:00:00Z","Data":146.93270414443987},{"Time":"2017-05-18T00:00:00Z","Data":150.10959762016597},{"Time":"2017-05-19T00:00:00Z","Data":152.39121326390136},{"Time":"2017-05-20T00:00:00Z","Data":180.13971628166118},{"Time":"2017-05-21T00:00:00Z","Data":174.0701273659106},{"Time":"2017-05-22T00:00:00Z","Data":159.99055800158052},{"Time":"2017-05-23T00:00:00Z","Data":158.92317846356926},{"Time":"2017-05-24T00:00:00Z","Data":153.22422962615363},{"Time":"2017-05-25T00:00:00Z","Data":154.99492395419225},{"Time":"2017-05-26T00:00:00Z","Data":159.89377321705084},{"Time":"2017-05-27T00:00:00Z","Data":190.37140086842882},{"Time":"2017-05-28T00:00:00Z","Data":192.39391276009522},{"Time":"2017-05-29T00:00:00Z","Data":221.53775725488637},{"Time":"2017-05-30T00:00:00Z","Data":157.6195511690753},{"Time":"2017-05-31T00:00:00Z","Data":153.51689355802245},{"Time":"2017-06-01T00:00:00Z","Data":161.01469414303875},{"Time":"2017-06-02T00:00:00Z","Data":159.41648382369135},{"Time":"2017-06-03T00:00:00Z","Data":184.85967566105822},{"Time":"2017-06-04T00:00:00Z","Data":177.16258618337514},{"Time":"2017-06-05T00:00:00Z","Data":170.6277157836757},{"Time":"2017-06-06T00:00:00Z","Data":166.50328652062203},{"Time":"2017-06-07T00:00:00Z","Data":156.40625786005333},{"Time":"2017-06-08T00:00:00Z","Data":153.99151680719737},{"Time":"2017-06-09T00:00:00Z","Data":155.3023113803091},{"Time":"2017-06-10T00:00:00Z","Data":172.9780689379916},{"Time":"2017-06-11T00:00:00Z","Data":172.61254312099592},{"Time":"2017-06-12T00:00:00Z","Data":163.0349783366797},{"Time":"2017-06-13T00:00:00Z","Data":164.64289712872394},{"Time":"2017-06-14T00:00:00Z","Data":153.18462778822897},{"Time":"2017-06-15T00:00:00Z","Data":161.51338722137433},{"Time":"2017-06-16T00:00:00Z","Data":163.09312911605588},{"Time":"2017-06-17T00:00:00Z","Data":189.79564677352576},{"Time":"2017-06-18T00:00:00Z","Data":169.85476443499823},{"Time":"2017-06-19T00:00:00Z","Data":160.63630076079437},{"Time":"2017-06-20T00:00:00Z","Data":155.87194704405388},{"Time":"2017-06-21T00:00:00Z","Data":149.14013904134652},{"Time":"2017-06-22T00:00:00Z","Data":154.48836350203177},{"Time":"2017-06-23T00:00:00Z","Data":159.08187521920425},{"Time":"2017-06-24T00:00:00Z","Data":187.8887943830156},{"Time":"2017-06-25T00:00:00Z","Data":172.6033807100661},{"Time":"2017-06-26T00:00:00Z","Data":153.45823409733003},{"Time":"2017-06-27T00:00:00Z","Data":153.0899410534322},{"Time":"2017-06-28T00:00:00Z","Data":145.12668618134757},{"Time":"2017-06-29T00:00:00Z","Data":149.8503305389005},{"Time":"2017-06-30T00:00:00Z","Data":159.7466572836031},{"Time":"2017-07-01T00:00:00Z","Data":186.1143798540833},{"Time":"2017-07-02T00:00:00Z","Data":187.08062626477792},{"Time":"2017-07-03T00:00:00Z","Data":174.21665303075972},{"Time":"2017-07-04T00:00:00Z","Data":198.02577286865994},{"Time":"2017-07-05T00:00:00Z","Data":150.4314360947944},{"Time":"2017-07-06T00:00:00Z","Data":150.77380861082565},{"Time":"2017-07-07T00:00:00Z","Data":151.79735898271926},{"Time":"2017-07-08T00:00:00Z","Data":180.6663918751816},{"Time":"2017-07-09T00:00:00Z","Data":173.26371527635473},{"Time":"2017-07-10T00:00:00Z","Data":155.56958030924582},{"Time":"2017-07-11T00:00:00Z","Data":162.52719145172472},{"Time":"2017-07-12T00:00:00Z","Data":149.6804156538082},{"Time":"2017-07-13T00:00:00Z","Data":150.50706822372464},{"Time":"2017-07-14T00:00:00Z","Data":153.18969389282847},{"Time":"2017-07-15T00:00:00Z","Data":180.3392547081608},{"Time":"2017-07-16T00:00:00Z","Data":194.594936109049},{"Time":"2017-07-17T00:00:00Z","Data":154.41438234247062},{"Time":"2017-07-18T00:00:00Z","Data":152.38948960726088},{"Time":"2017-07-19T00:00:00Z","Data":147.15870500339597},{"Time":"2017-07-20T00:00:00Z","Data":150.3863681394888},{"Time":"2017-07-21T00:00:00Z","Data":162.94423682860702},{"Time":"2017-07-22T00:00:00Z","Data":185.90346846763728},{"Time":"2017-07-23T00:00:00Z","Data":175.70127090554388},{"Time":"2017-07-24T00:00:00Z","Data":162.71704195129263},{"Time":"2017-07-25T00:00:00Z","Data":155.42196025017807},{"Time":"2017-07-26T00:00:00Z","Data":154.38483777706395},{"Time":"2017-07-27T00:00:00Z","Data":156.25826471780948},{"Time":"2017-07-28T00:00:00Z","Data":154.3498053193344},{"Time":"2017-07-29T00:00:00Z","Data":181.74308386194255},{"Time":"2017-07-30T00:00:00Z","Data":171.82748133812552},{"Time":"2017-07-31T00:00:00Z","Data":152.7690096065352},{"Time":"2017-08-01T00:00:00Z","Data":153.69817128372262},{"Time":"2017-08-02T00:00:00Z","Data":150.33955422999043},{"Time":"2017-08-03T00:00:00Z","Data":153.15799390103487},{"Time":"2017-08-04T00:00:00Z","Data":162.63429836791602},{"Time":"2017-08-05T00:00:00Z","Data":192.2913971630805},{"Time":"2017-08-06T00:00:00Z","Data":187.86931309486096},{"Time":"2017-08-07T00:00:00Z","Data":168.21201548535106},{"Time":"2017-08-08T00:00:00Z","Data":156.59857079118484},{"Time":"2017-08-09T00:00:00Z","Data":150.7896743274153},{"Time":"2017-08-10T00:00:00Z","Data":152.8856153747525},{"Time":"2017-08-11T00:00:00Z","Data":159.46105453495235},{"Time":"2017-08-12T00:00:00Z","Data":186.08437186243202},{"Time":"2017-08-13T00:00:00Z","Data":179.73892284977026},{"Time":"2017-08-14T00:00:00Z","Data":160.0241634866143},{"Time":"2017-08-15T00:00:00Z","Data":158.8828255267546},{"Time":"2017-08-16T00:00:00Z","Data":156.5593355330527},{"Time":"2017-08-17T00:00:00Z","Data":158.13846655404404},{"Time":"2017-08-18T00:00:00Z","Data":160.15395271638982},{"Time":"2017-08-19T00:00:00Z","Data":184.01455455912205},{"Time":"2017-08-20T00:00:00Z","Data":179.42750595628604},{"Time":"2017-08-21T00:00:00Z","Data":153.4151894617011},{"Time":"2017-08-22T00:00:00Z","Data":158.6379466768123},{"Time":"2017-08-23T00:00:00Z","Data":147.04706476074375},{"Time":"2017-08-24T00:00:00Z","Data":149.91985540992474},{"Time":"2017-08-25T00:00:00Z","Data":160.67241168632427},{"Time":"2017-08-26T00:00:00Z","Data":178.92645303893386},{"Time":"2017-08-27T00:00:00Z","Data":171.59350285837934},{"Time":"2017-08-28T00:00:00Z","Data":146.91614691614691},{"Time":"2017-08-29T00:00:00Z","Data":147.3259486497514},{"Time":"2017-08-30T00:00:00Z","Data":141.54639250748738},{"Time":"2017-08-31T00:00:00Z","Data":146.9509779202536},{"Time":"2017-09-01T00:00:00Z","Data":156.9978136512025},{"Time":"2017-09-02T00:00:00Z","Data":193.45721561224042},{"Time":"2017-09-03T00:00:00Z","Data":186.04084355277337},{"Time":"2017-09-04T00:00:00Z","Data":208.78075760846104},{"Time":"2017-09-05T00:00:00Z","Data":149.85482885085574},{"Time":"2017-09-06T00:00:00Z","Data":149.20673185578036},{"Time":"2017-09-07T00:00:00Z","Data":137.43206772893637},{"Time":"2017-09-08T00:00:00Z","Data":135.94691882676872},{"Time":"2017-09-09T00:00:00Z","Data":152.31370916500515},{"Time":"2017-09-10T00:00:00Z","Data":146.67331380735754},{"Time":"2017-09-11T00:00:00Z","Data":131.20819365702192},{"Time":"2017-09-12T00:00:00Z","Data":136.58614694848566},{"Time":"2017-09-13T00:00:00Z","Data":136.43513950262417},{"Time":"2017-09-14T00:00:00Z","Data":140.03273367442372},{"Time":"2017-09-15T00:00:00Z","Data":148.45248499018263},{"Time":"2017-09-16T00:00:00Z","Data":157.82168041493819},{"Time":"2017-09-17T00:00:00Z","Data":154.30347058656602},{"Time":"2017-09-18T00:00:00Z","Data":133.34619581126753},{"Time":"2017-09-19T00:00:00Z","Data":134.33724461808112},{"Time":"2017-09-20T00:00:00Z","Data":128.36482084690553},{"Time":"2017-09-21T00:00:00Z","Data":127.3566174384963},{"Time":"2017-09-22T00:00:00Z","Data":134.71363127783994},{"Time":"2017-09-23T00:00:00Z","Data":154.7964382098685},{"Time":"2017-09-24T00:00:00Z","Data":149.68455105975326},{"Time":"2017-09-25T00:00:00Z","Data":131.03715644065375},{"Time":"2017-09-26T00:00:00Z","Data":130.9769379240815},{"Time":"2017-09-27T00:00:00Z","Data":124.03553134928353},{"Time":"2017-09-28T00:00:00Z","Data":129.30495925989925},{"Time":"2017-09-29T00:00:00Z","Data":137.477077602734},{"Time":"2017-09-30T00:00:00Z","Data":154.86411111718706},{"Time":"2017-10-01T00:00:00Z","Data":146.37159032424086},{"Time":"2017-10-02T00:00:00Z","Data":130.60473363243156},{"Time":"2017-10-03T00:00:00Z","Data":128.31398145860692},{"Time":"2017-10-04T00:00:00Z","Data":127.71896425309681},{"Time":"2017-10-05T00:00:00Z","Data":126.76503153730359},{"Time":"2017-10-06T00:00:00Z","Data":126.58040743497564},{"Time":"2017-10-07T00:00:00Z","Data":149.49487159713118},{"Time":"2017-10-08T00:00:00Z","Data":145.60991860027232},{"Time":"2017-10-09T00:00:00Z","Data":138.50915821047928},{"Time":"2017-10-10T00:00:00Z","Data":125.31807883704398},{"Time":"2017-10-11T00:00:00Z","Data":124.87681353321629},{"Time":"2017-10-12T00:00:00Z","Data":124.94225889529402},{"Time":"2017-10-13T00:00:00Z","Data":123.05725379604979},{"Time":"2017-10-14T00:00:00Z","Data":146.1753312847626},{"Time":"2017-10-15T00:00:00Z","Data":140.6601164911455},{"Time":"2017-10-16T00:00:00Z","Data":125.76987355763147},{"Time":"2017-10-17T00:00:00Z","Data":125.70419965096913},{"Time":"2017-10-18T00:00:00Z","Data":119.2393307484494},{"Time":"2017-10-19T00:00:00Z","Data":124.92608828626771},{"Time":"2017-10-20T00:00:00Z","Data":121.80356211751146},{"Time":"2017-10-21T00:00:00Z","Data":144.17288648226742},{"Time":"2017-10-22T00:00:00Z","Data":144.9383863936871},{"Time":"2017-10-23T00:00:00Z","Data":133.82708857774912},{"Time":"2017-10-24T00:00:00Z","Data":128.37054071104419},{"Time":"2017-10-25T00:00:00Z","Data":121.6326902074991},{"Time":"2017-10-26T00:00:00Z","Data":121.423755549619},{"Time":"2017-10-27T00:00:00Z","Data":123.2875088181069},{"Time":"2017-10-28T00:00:00Z","Data":134.1320125704841},{"Time":"2017-10-29T00:00:00Z","Data":130.89785689595575},{"Time":"2017-10-30T00:00:00Z","Data":120.12238246056395},{"Time":"2017-10-31T00:00:00Z","Data":103.77387271950698},{"Time":"2017-11-01T00:00:00Z","Data":121.42862982007242},{"Time":"2017-11-02T00:00:00Z","Data":119.85197737658143},{"Time":"2017-11-03T00:00:00Z","Data":126.00143082662757},{"Time":"2017-11-04T00:00:00Z","Data":146.40639856815258},{"Time":"2017-11-05T00:00:00Z","Data":144.52935468219312},{"Time":"2017-11-06T00:00:00Z","Data":123.43885223384069},{"Time":"2017-11-07T00:00:00Z","Data":124.72662921261485},{"Time":"2017-11-08T00:00:00Z","Data":126.49875574994343},{"Time":"2017-11-09T00:00:00Z","Data":127.6953454763983},{"Time":"2017-11-10T00:00:00Z","Data":136.36646226015202},{"Time":"2017-11-11T00:00:00Z","Data":154.57074601590614},{"Time":"2017-11-12T00:00:00Z","Data":147.4578207204241},{"Time":"2017-11-13T00:00:00Z","Data":126.2963544940289},{"Time":"2017-11-14T00:00:00Z","Data":124.28645961207954},{"Time":"2017-11-15T00:00:00Z","Data":117.14067034957299},{"Time":"2017-11-16T00:00:00Z","Data":122.29454654184971},{"Time":"2017-11-17T00:00:00Z","Data":127.30744374596087},{"Time":"2017-11-18T00:00:00Z","Data":156.8434436651491},{"Time":"2017-11-19T00:00:00Z","Data":160.22898842476093},{"Time":"2017-11-20T00:00:00Z","Data":135.78907650221677},{"Time":"2017-11-21T00:00:00Z","Data":133.83531240987574},{"Time":"2017-11-22T00:00:00Z","Data":151.64344546224308},{"Time":"2017-11-23T00:00:00Z","Data":539.6081165689567},{"Time":"2017-11-24T00:00:00Z","Data":384.5590951717867},{"Time":"2017-11-25T00:00:00Z","Data":243.16268727831013},{"Time":"2017-11-26T00:00:00Z","Data":195.72620859105848},{"Time":"2017-11-27T00:00:00Z","Data":198.0795306749315},{"Time":"2017-11-28T00:00:00Z","Data":147.0323807277926},{"Time":"2017-11-29T00:00:00Z","Data":131.6886099535616},{"Time":"2017-11-30T00:00:00Z","Data":133.63038315223102},{"Time":"2017-12-01T00:00:00Z","Data":131.2217194570136},{"Time":"2017-12-02T00:00:00Z","Data":159.9936870555046},{"Time":"2017-12-03T00:00:00Z","Data":153.6494552167595},{"Time":"2017-12-04T00:00:00Z","Data":135.96184651230521},{"Time":"2017-12-05T00:00:00Z","Data":134.1348835843683},{"Time":"2017-12-06T00:00:00Z","Data":135.15052605993264},{"Time":"2017-12-07T00:00:00Z","Data":145.69132292031836},{"Time":"2017-12-08T00:00:00Z","Data":154.10480465326287},{"Time":"2017-12-09T00:00:00Z","Data":198.23959859812862},{"Time":"2017-12-10T00:00:00Z","Data":189.86650238575405},{"Time":"2017-12-11T00:00:00Z","Data":167.28721448672783},{"Time":"2017-12-12T00:00:00Z","Data":163.28448954013948},{"Time":"2017-12-13T00:00:00Z","Data":160.11282561805209}],"Meta":{"VendorCode":"Number of Visits","Label":"Number of Visits","Units":"#","Source":"PlaceIQ","Upsample":4,"Downsample":3,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2016-10-01T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"BestBuy","UniqueId":"BestBuy","IsCustom":true}}],"Title":"BestBuy → Number of Visits All Available","Error":"","GraphicalPreference":""}
`

var bogusData2 = `{"EntityData":[{"Data":[{"Data":[{"Time":"2016-11-01T00:00:00Z","Data":1},{"Time":"2017-11-01T00:00:00Z","Data":1}],"Meta":{"VendorCode":"weight","Label":"Weights","Units":"Weight","Source":"Third-Party","Upsample":4,"Downsample":1,"IsTransformed":false},"IsWeight":true}],"Category":{"Data":[{"Time":"2016-11-01T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"BestBuy","UniqueId":"BestBuy","IsCustom":true}}],"Title":"BestBuy → Quarter Dates All Available","Error":"","GraphicalPreference":""}`

var bogusData3 = `{"EntityData":[{"Data":[{"Data":[{"Time":"2017-11-01T00:00:00Z","Data":0.01877},{"Time":"2017-11-03T00:00:00Z","Data":0.01867},{"Time":"2017-11-09T00:00:00Z","Data":0.01867},{"Time":"2017-11-13T00:00:00Z","Data":0.01867},{"Time":"2017-11-16T00:00:00Z","Data":0.02283},{"Time":"2017-11-17T00:00:00Z","Data":0.02283},{"Time":"2017-11-29T00:00:00Z","Data":0.02283},{"Time":"2018-01-08T00:00:00Z","Data":0.02283},{"Time":"2018-01-19T00:00:00Z","Data":0.02283}],"Meta":{"VendorCode":"Same-Store Sales Street Est.","Label":"Same-Store Sales Street Est.","Units":"%","Source":"Third-Party","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-11-01T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"BestBuy","UniqueId":"BestBuy","IsCustom":true}}],"Title":"BestBuy → SSS Estimate All Available","Error":"","GraphicalPreference":""}
`

var sqlDb *sql.DB

func connect() (*sql.DB, error) {
	var db *sql.DB

	connectionName := os.Getenv("CLOUDSQL_CONNECTION_NAME")
	user := os.Getenv("CLOUDSQL_USER")
	password := os.Getenv("CLOUDSQL_PASSWORD") // NOTE: password may be empty

	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@cloudsql(%s)/location", user, password, connectionName))
	if err != nil {
		return nil, err
	}

	// rows, err := db.Query("SHOW DATABASES")
	// if err != nil {
	// 	return nil, err
	// }
	// defer rows.Close()

	return db, nil
}

func QueryStringArray(ctx context.Context, query string, args ...interface{}) []string {
	var err error
	if sqlDb == nil {
		sqlDb, err = connect()
	}
	if err != nil {
		log.Errorf(ctx, "Unable to connect to database")
	}

	sArr := make([]string, 0)
	rows, err := sqlDb.Query(query, args...)

	if err != nil {
		log.Errorf(ctx, "query error = %s", err)
		return sArr
	}

	var s string

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&s)

		if err != nil {
			log.Errorf(ctx, "query error = %s", err)
		} else {
			sArr = append(sArr, s)
		}
	}

	return sArr
}

func GenericQuery(ctx context.Context, query string, metaMap map[string]SeriesMeta, stringJoiner string, isWeight bool) MultiEntityData {
	var err error
	if sqlDb == nil {
		sqlDb, err = connect()
	}
	if err != nil {
		log.Errorf(ctx, "Unable to connect to database")
	}

	var m MultiEntityData

	rows, err := sqlDb.Query(query)

	if err != nil {
		log.Errorf(ctx, "query error = %s", err)
		return m
	}

	m.Title = ""
	var date, entity, field, subfield string
	var data float64

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&date, &entity, &field, &subfield, &data)

		if err != nil {
			log.Errorf(ctx, "query error = %s", err)
		} else {
			t, _ := time.Parse("2006-01-02", date)

			seriesMeta := metaMap[field]

			if subfield != "" {
				seriesMeta.VendorCode = seriesMeta.VendorCode + stringJoiner + subfield
				seriesMeta.Label = seriesMeta.VendorCode
			}

			m = m.Insert(
				EntityMeta{
					Name:     entity,
					UniqueId: entity,
					IsCustom: true,
				},
				seriesMeta,
				DataPoint{
					Time: t,
					Data: data,
				},
				CategoryLabel{
					Id:    0,
					Label: "",
				},
				isWeight)
		}
	}

	return m
}

func listOfStringsToQuotedCommaList(entities []string) string {
	for i, v := range entities {
		entities[i] = "'" + strings.Replace(v, "'", "\\'", -1) + "'"
	}

	return strings.Join(entities, ", ")
}

func GetSQLData(ctx context.Context, entities []string, queryName string, parameters map[string][]string, startDate time.Time, endDate time.Time) MultiEntityData {
	cache := cache.NewGenericCache(
		time.Hour*24,
		"SQLData",
		func(ctx context.Context, key string) (interface{}, bool) {
			return GetSQLDataLive(ctx, entities, queryName, parameters, startDate, endDate), true
		})

	dmaList := parameters["DMA"]
	source := parameters["Movement Source"]
	confidence := parameters["Confidence Score"]
	norm := parameters["Normalization"]

	iface, found := cache.Retrieve(ctx, queryName+":"+fmt.Sprintf("%s", entities)+":"+fmt.Sprintf("%s", dmaList)+":"+fmt.Sprintf("%s", source)+":"+fmt.Sprintf("%s", confidence)+":"+fmt.Sprintf("%s", norm))

	if found {
		switch m := iface.(type) {
		case MultiEntityData:
			return m
		}
	}

	return MultiEntityData{Error: "Unable to retrieve from SQL Data"}
}

func GetSQLDataLive(ctx context.Context, entities []string, queryName string, parameters map[string][]string, startDate time.Time, endDate time.Time) MultiEntityData {
	timer := time.Now()

	metaMap := map[string]SeriesMeta{
		"Number of Visits": SeriesMeta{
			VendorCode:    "Number of Visits",
			Label:         "Number of Visits",
			Units:         "#",
			Source:        "PlaceIQ",
			Upsample:      ResampleZero,
			Downsample:    ResampleArithmetic,
			IsTransformed: false,
		},
		"% of Traffic": SeriesMeta{
			VendorCode:    "% of Traffic",
			Label:         "% of Traffic",
			Units:         "%",
			Source:        "PlaceIQ",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
		"SSS Estimate": SeriesMeta{
			VendorCode:    "Same-Store Sales Street Est.",
			Label:         "Same-Store Sales Street Est.",
			Units:         "%",
			Source:        "Third-Party",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
		"WoW Change": SeriesMeta{
			VendorCode:    "WoW Change",
			Label:         "WoW Change",
			Units:         "%",
			Source:        "Narrative",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
		"weight": SeriesMeta{
			VendorCode:    "weight",
			Label:         "weight",
			Units:         "weight",
			Source:        "Third-Party",
			Upsample:      ResampleZero,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
	}

	var dateRestriction string
	if !startDate.IsZero() {
		dateRestriction = " and local_date >= '" + startDate.String() + "' "
	} else {
		dateRestriction = ""
	}

	var dmaRestriction string
	var dmaList []string
	var normalization string = "raw"
	var movementSource string = "both"
	var dataTable string = "location.with_store_count"
	if parameters != nil {
		dmaList = parameters["DMA"]
		if len(dmaList) > 0 {
			dmaRestriction = " and dma in (" + listOfStringsToQuotedCommaList(dmaList) + ")"
		} else {
			dmaRestriction = ""
		}

		source := parameters["Movement Source"]
		if len(source) > 0 {
			switch source[0] {
			case "Foreground":
				movementSource = "foreground"
			case "Background":
				movementSource = "background"
			default:
				movementSource = "both"
			}
		}

		confidence := parameters["Confidence Score"]
		if len(confidence) > 0 {
			switch confidence[0] {
			case "zero":
				dataTable = "location.with_store_count"
			case "zero_point_two":
				dataTable = "location.with_02_filter"
			default:
				dataTable = "location.with_store_count"
			}
		}

		norm := parameters["Normalization"]
		if len(norm) > 0 {
			switch norm[0] {
			case "raw":
				normalization = "raw"
			case "placeiq":
				if movementSource == "foreground" {
					normalization = "placeiq_fg"
				} else if movementSource == "background" {
					normalization = "placeiq_bg"
				} else {
					normalization = "placeiq_all"
				}
			case "alphahat_raw":
				if movementSource == "foreground" {
					normalization = "ah_fg*1000"
				} else if movementSource == "background" {
					normalization = "ah_bg*1000"
				} else {
					normalization = "ah_all*1000"
				}
			case "alphahat_smoothed":
				if movementSource == "foreground" {
					normalization = "smoothed_fg*1000"
				} else if movementSource == "background" {
					normalization = "smoothed_bg*1000"
				} else {
					normalization = "smoothed_all*1000"
				}
			default:
				normalization = "raw"
			}
		}
	}

	var movementSourceQuery string
	if movementSource == "foreground" {
		movementSourceQuery = `where movement_source = 'foreground'`
	} else if movementSource == "background" {
		movementSourceQuery = `where movement_source = 'background'`
	} else {
		movementSourceQuery = `where movement_source in ('background', 'foreground')`
	}

	var query string
	var isWeight bool = false
	switch queryName {
	case "Number of Visits":
		if len(dmaList) > 0 {
			query = `SELECT date_format(local_date, '%Y-%m-%d') as date, brand, "Number of Visits" as field, dma as subfield, sum(visit_count) as visit_count FROM ` + dataTable + `
			` + movementSourceQuery + `
			and brand in (` + listOfStringsToQuotedCommaList(entities) + `) ` + dateRestriction + dmaRestriction + `
			group by local_date, brand, dma`
		} else {
			query = `SELECT date_format(local_date, '%Y-%m-%d') as date, brand, "Number of Visits" as field, "" as subfield, sum(visit_count) as visit_count FROM ` + dataTable + `
			` + movementSourceQuery + `
			and brand in (` + listOfStringsToQuotedCommaList(entities) + `) ` + dateRestriction + dmaRestriction + `
			group by local_date, brand`
		}

		query = `SELECT a.date, a.brand, a.field, a.subfield, a.visit_count / b.` + normalization + ` as normalized_visit_count FROM ( ` +
			query +
			`) a JOIN location.factors b on a.date = b.date `

	case "Traffic By DMA":
		query = `SELECT date_format(local_date, '%Y-%m-%d') as date, brand, "Number of Visits" as field, dma as subfield, sum(visit_count) as visit_count FROM ` + dataTable + `
		` + movementSourceQuery + `
		and brand in (` + listOfStringsToQuotedCommaList(entities) + `) ` + dateRestriction + `
		group by local_date, brand, dma`

		query = `SELECT a.date, a.subfield as brand, a.field, '' as subfield, a.visit_count / b.` + normalization + ` as normalized_visit_count FROM ( ` +
			query +
			`) a JOIN location.factors b on a.date = b.date `

	case "Traffic Contribution":
		dateRestriction = ` and local_date = (select max(local_date) from ` + dataTable + `) `

		entityList := listOfStringsToQuotedCommaList(entities)

		query1 := `SELECT date_format(local_date, '%Y-%m-%d') as date, brand, "Number of Visits" as field, dma as subfield, sum(visit_count) as visit_count FROM ` + dataTable + `
		` + movementSourceQuery + `
		and brand in (` + entityList + `) ` + dateRestriction + `
		group by local_date, brand, dma`

		query2 := `SELECT date_format(local_date, '%Y-%m-%d') as date, brand, "Number of Visits" as field, '' as subfield, sum(visit_count) as visit_count FROM ` + dataTable + `
		` + movementSourceQuery + `
		and brand in (` + entityList + `) ` + dateRestriction + `
		group by local_date, brand`

		query = `SELECT a.date, a.subfield as brand, '% of Traffic' as field, '' as subfield, a.visit_count / b.visit_count as value
			FROM (` +
			query1 +
			`) a LEFT JOIN
			(` +
			query2 +
			`) b ON a.brand = b.brand `

	case "SSS Estimate":
		query = `SELECT date_format(date, '%Y-%m-%d') as date, brand, 'SSS Estimate' as field, '' as subfield, sss FROM location.sss
		where brand in (` + listOfStringsToQuotedCommaList(entities) + `) ` + dateRestriction
	case "Quarter Dates":
		query = `SELECT date_format(date, '%Y-%m-%d') as date, brand, 'weight' as field, '' as subfield, 1 as value
			FROM location.quarter_dates
			where brand in (` + listOfStringsToQuotedCommaList(entities) + `) `
		isWeight = true
	}

	log.Infof(ctx, "running query = %s", query)

	if appengine.IsDevAppServer() {
		time.Sleep(time.Second * 1)
		var m MultiEntityData
		log.Infof(ctx, "CANNOT DO A SQL CONNECT ON THE DEV SERVER")
		switch queryName {
		case "Number of Visits":
			json.Unmarshal([]byte(bogusData), &m)
		case "Traffic By DMA":
			json.Unmarshal([]byte(bogusData), &m)
		case "Traffic Contribution":
			json.Unmarshal([]byte(bogusData), &m)
		case "Quarter Dates":
			json.Unmarshal([]byte(bogusData2), &m)
		case "SSS Estimate":
			json.Unmarshal([]byte(bogusData3), &m)
		default:
			json.Unmarshal([]byte(bogusData), &m)
		}
		return m
	}

	temp := GenericQuery(ctx, query, metaMap, " in ", isWeight)
	log.Infof(ctx, "time taken was %v\nentities = %s\nqueryName = %s\nparameters = %s\nstartDate = %s\nendDate = %s", time.Since(timer), entities, queryName, parameters, startDate, endDate)

	return temp
}

// func GetVisitCounts(ctx context.Context) MultiEntityData {
// 	return GenericQuery(ctx, `select date, brand, "Number of Visits" as field, sum(num_visit)
// from wow_change
// group by date, brand`,
// 		map[string]SeriesMeta{
// 			"Number of Visits": SeriesMeta{
// 				VendorCode:    "Number of Visits",
// 				Label:         "Number of Visits",
// 				Units:         "#",
// 				Source:        "Narrative",
// 				Upsample:      ResampleZero,
// 				Downsample:    ResampleArithmetic,
// 				IsTransformed: false,
// 			},
// 		}, "")
//
// }
