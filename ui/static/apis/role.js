const roleApi = {
    List(page,pageSize){
        return request.get(`/role/list?page=${page}&pageSize=${pageSize}`);
    },
    Delete(id){
        let params = {id:id};
        return request.post(`/role/delete`,params);
    },
    Edit(id,data){
        let ndata = {};
        ndata._id = id;
        ndata.roles = JSON.stringify(data.roles);
        return request.post(`/role/edit`,ndata);
    },
    Add(data){
        let ndata = {};
        ndata.name = data.name;
        ndata.roles = JSON.stringify(data.roles);
        return request.post("/role/add",ndata);
    },
    GetLang(path,objs){
        let result = _.split(path,".");
        let tag = {};

        function search(arr,obj) {
            for (const item of arr) {
                let v1 = _.find(obj,function (v) {
                    return v.mark === item;
                })
                if(v1 === undefined){
                    break;
                }
                tag = v1
                if(('children' in v1) && v1.children !== undefined){
                    search(_.tail(arr),v1.children);
                }
            }
        }
        search(result,objs);
        return tag;
    },
    GetId(path){
        return this.GetLang(path,Nav)?.id;
    },
    TileTree(tree){
        return _.flatMap(tree,(node)=>{
            let children = node.children ? roleApi.TileTree(node.children) : [];
            return [node,...children];
        });
    }
}