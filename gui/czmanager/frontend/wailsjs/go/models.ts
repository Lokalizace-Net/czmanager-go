export namespace main {
	
	export class DetectedGame {
	    name: string;
	    path: string;
	    platform: string;
	    appId?: string;
	
	    static createFrom(source: any = {}) {
	        return new DetectedGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.platform = source["platform"];
	        this.appId = source["appId"];
	    }
	}

}

