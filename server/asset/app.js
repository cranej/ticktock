const {createApp} = Vue
createApp({
    data() {
        return {
            recentTitles: [],
            detailObject: null ,
            ongoing: null,
            error: null,
            newStart: '',
            report: null,
            queryParam: {'dayStart': "", "dayEnd": "", 'viewType': "summary"},
        }
    },

    created() {
        this.getData();
        // default value for queryParam.dayStart/dayEnd
        let today = this.getDateString(new Date());
        this.queryParam.dayStart = today;
        this.queryParam.dayEnd = today;
    },

    methods: {
        async getRecent() {
            const url = '/api/recent/';
            this.recentTitles = await (await fetch(url)).json()
        },
        async getUnfinished() {
            const url = '/api/unfinished/';
            this.ongoing = await (await fetch(url)).json();
        },

        getData() {
            this.getRecent();
            this.getUnfinished();
            this.error = null;
        },
        async start(title) {
            if (title == null || title.length == 0) {
                this.error = "Empty title";
                return;
            }

            let url = `/api/start/${encodeURI(title)}`;
            await (fetch(url, {method: 'POST'})
                   .then((rep) => {
                       if (rep.ok) {
                           this.getData();
                       } else {
                           this.error = `${rep.status}`;
                       }
                   }).catch((err) => this.error = err))
        },
        async finish() {
            let url = `/api/finish`;
            await (fetch(url, {method: 'POST', body: this.ongoing.Notes})
                   .then((rep) => {
                       if (rep.ok) {
                           this.getData();
                       } else {
                           this.error = `${rep.status}`;
                       }
                   }).catch((err) => this.error = err))
        },
        async getReportByDate(dayStart, dayEnd, viewType) {
            this.report = null;
            if (dayStart == "" || dayEnd == "") {
                this.error = "Query start and end must be specified.";
            } else {
                let url = `/api/report-by-date/${dayStart}/${dayEnd}?view_type=${viewType}`;
                this.report = await (await fetch(url)).text();
            }
        },
        async getItemDetail(title) {
          let url = `/api/latest/${encodeURI(title)}`;
          this.detailObject = await (await fetch(url)).text();
          if (this.detailObject == "") {
            this.detailObject = null;
          }
        },
        onQuickReport(offset, days) {
            let one_day = 86400000;
            let now_t = new Date().getTime();
            let start_t = now_t - (parseInt(offset, 10) * one_day);
            let end_t = days == 'null' ? now_t : (start_t + ((parseInt(days, 10) - 1) * one_day));

            this.queryParam.dayStart = this.getDateString(new Date(start_t));
            this.queryParam.dayEnd = this.getDateString(new Date(end_t));
            this.getReportByDate(this.queryParam.dayStart, this.queryParam.dayEnd, this.queryParam.viewType);
        },
        getDateString(d) {
            return `${d.getFullYear()}-${(d.getMonth() + 1).toString().padStart(2, '0')}-${d.getDate().toString().padStart(2, '0')}`;
        }
    }
}).mount("#layout");
