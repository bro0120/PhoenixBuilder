const fs=require("fs");
function doit(dir) {
	let dir_content=fs.readdirSync(dir);
	for(let item of dir_content) {
		if(item.indexOf(".js")!=-1)continue;
		let fn=`${dir}/${item}`;
		if(fs.lstatSync(fn).isDirectory()) {
			doit(fn);
			continue;
		}else{
			let fc=fs.readFileSync(fn).toString();
			fs.writeFileSync(fn, fc.replace(/github.com\/sandertv\/gophertunnel/g,"phoenixbuilder"));
		}
	}
}

doit(".");