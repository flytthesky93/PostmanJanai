<script setup>
defineProps({
  data: String
})
</script>

<template>
  <div class="flex-1 flex flex-col p-4 bg-[#181818] overflow-hidden">
    <div class="flex gap-4 text-xs font-bold border-b border-gray-800 mb-4 pb-2">
      <span class="text-white border-b-2 border-orange-500 pb-2 cursor-pointer">Body</span>
      <span class="text-gray-500 cursor-pointer hover:text-gray-300">Headers</span>
    </div>
    <div class="flex-1 bg-[#121212] rounded border border-gray-800 p-4 font-mono text-xs overflow-auto shadow-inner">
      <pre v-if="data" class="text-blue-400 leading-relaxed">{{ data }}</pre>
      <div v-else class="h-full flex items-center justify-center text-gray-600 italic">
        Nhấn "Send" để xem kết quả...
      </div>
    </div>
  </div>
</template>