export namespace main {
	
	export class AgentStatus {
	    running: boolean;
	    version: string;
	    busy: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.version = source["version"];
	        this.busy = source["busy"];
	    }
	}
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
	export class LoginResult {
	    accessToken: string;
	    refreshToken: string;
	    expiresAt: string;
	    refreshExpiresAt: string;
	    user: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new LoginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.expiresAt = source["expiresAt"];
	        this.refreshExpiresAt = source["refreshExpiresAt"];
	        this.user = source["user"];
	    }
	}

}

