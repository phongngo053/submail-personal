export namespace icon {
	
	export class Preset {
	    name: string;
	    plan: string;
	    category: string;
	    price: number;
	    cycle: string;
	    payment: string;
	    iconBg: string;
	    iconText: string;
	    prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.plan = source["plan"];
	        this.category = source["category"];
	        this.price = source["price"];
	        this.cycle = source["cycle"];
	        this.payment = source["payment"];
	        this.iconBg = source["iconBg"];
	        this.iconText = source["iconText"];
	        this.prompt = source["prompt"];
	    }
	}

}

export namespace integrations {
	
	export class View {
	    id: string;
	    name: string;
	    type: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new View(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	    }
	}

}

export namespace main {
	
	export class AccountView {
	    email: string;
	    tag: string;
	    color: string;
	    initials: string;
	    password: string;
	    secret2fa: string;
	    recoveryEmail: string;
	    phone: string;
	    appPassword: string;
	    note: string;
	    backupCodes: string[];
	    iconFile: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.tag = source["tag"];
	        this.color = source["color"];
	        this.initials = source["initials"];
	        this.password = source["password"];
	        this.secret2fa = source["secret2fa"];
	        this.recoveryEmail = source["recoveryEmail"];
	        this.phone = source["phone"];
	        this.appPassword = source["appPassword"];
	        this.note = source["note"];
	        this.backupCodes = source["backupCodes"];
	        this.iconFile = source["iconFile"];
	    }
	}
	export class IconResult {
	    file: string;
	    dataUri: string;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new IconResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.dataUri = source["dataUri"];
	        this.bytes = source["bytes"];
	    }
	}
	export class SubscriptionView {
	    id: number;
	    name: string;
	    plan: string;
	    category: string;
	    gmail: string;
	    price: number;
	    cycle: string;
	    nextDate: string;
	    nextText: string;
	    urgency: string;
	    payment: string;
	    receiptId: string;
	    status: string;
	    authMode: string;
	    customPass: string;
	    iconBg: string;
	    iconText: string;
	    iconFile: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.plan = source["plan"];
	        this.category = source["category"];
	        this.gmail = source["gmail"];
	        this.price = source["price"];
	        this.cycle = source["cycle"];
	        this.nextDate = source["nextDate"];
	        this.nextText = source["nextText"];
	        this.urgency = source["urgency"];
	        this.payment = source["payment"];
	        this.receiptId = source["receiptId"];
	        this.status = source["status"];
	        this.authMode = source["authMode"];
	        this.customPass = source["customPass"];
	        this.iconBg = source["iconBg"];
	        this.iconText = source["iconText"];
	        this.iconFile = source["iconFile"];
	    }
	}
	export class Snapshot {
	    gmailAccounts: AccountView[];
	    subscriptionsData: SubscriptionView[];
	    locked: boolean;
	    vndRate: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gmailAccounts = this.convertValues(source["gmailAccounts"], AccountView);
	        this.subscriptionsData = this.convertValues(source["subscriptionsData"], SubscriptionView);
	        this.locked = source["locked"];
	        this.vndRate = source["vndRate"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TotpResult {
	    code: string;
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new TotpResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.remaining = source["remaining"];
	    }
	}

}

