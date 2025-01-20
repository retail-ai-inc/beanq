<template>
  <div>
    <div id="payloadAlertInfo">
      <div class="alert alert-dismissible" :class="type" role="alert" v-if="showAlert">
        <div>{{message}}</div>
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
      </div>
    </div>
    <div ref="editor"></div>
  </div>
</template>

<script setup>
import { ref,defineProps,defineEmits, onMounted,onUnmounted,watch } from 'vue';

const props = defineProps({
  data:{}
})

const [editor,editorContent] = [ref(null),ref(props.data)];
let [codeMirrorInstance,codeTimer] = [null,null];
const [type,message,showAlert] = [ref(""),ref(""),ref(false)];
// init CodeMirror
onMounted(() => {
  try {
    const formateVal = JSON.stringify(JSON.parse(props.data),null,3);
    codeTimer = setTimeout(()=>{
      codeMirrorInstance = CodeMirror(editor.value, {
        mode: 'javascript',
        lineNumbers: true,
        theme: 'dracula',
        value: formateVal ,
      });

      codeMirrorInstance.on("change",function (editor,obj){
        editorContent.value = codeMirrorInstance.getValue();
      })

    },500);

  }catch (e) {
    console.log(e)
  }
});

const emits = defineEmits(["getValue"]);

watch(
    ()=>editorContent.value,
    (n,o)=>{

      showAlert.value = false;
      let newjson = {};
      try {
        newjson = JSON.parse(n);
      }catch (e) {
        showAlert.value = true;
        type.value = "alert-danger";
        message.value = "Must be in JSON format";
      }
      if(showAlert.value === false){
        emits("getValue",newjson);
      }
  }
)

onUnmounted(()=>{
  clearInterval(codeTimer);
})

</script>

<style>
</style>