<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑设备' : '添加设备'"
    width="560px"
    @closed="resetForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="设备名称" />
      </el-form-item>
      <el-form-item label="协议" prop="protocol">
        <el-select v-model="form.protocol" style="width: 100%">
          <el-option label="OPC UA" value="opcua" />
          <el-option label="Modbus TCP" value="modbus" />
          <el-option label="MQTT" value="mqtt" />
          <el-option label="S7" value="s7" />
          <el-option label="BACnet/IP" value="bacnet" />
        </el-select>
      </el-form-item>
      <el-form-item label="地址" prop="address">
        <el-input v-model="form.address" :placeholder="addressPlaceholder" />
      </el-form-item>
      <el-form-item label="配置" prop="config">
        <el-input v-model="form.config" type="textarea" :rows="4" placeholder='JSON 字符串，如 {"unit_id":1}' />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="2" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { createDevice, updateDevice } from '@/api/device'
import type { Device, DeviceForm } from '@/types'

const emit = defineEmits<{ saved: [] }>()

const visible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const defaultForm = (): DeviceForm => ({
  name: '',
  protocol: 'modbus',
  address: '127.0.0.1:502',
  config: '{"unit_id":1}',
  description: '',
})

const form = reactive<DeviceForm>(defaultForm())

const rules: FormRules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择协议', trigger: 'change' }],
  address: [{ required: true, message: '请输入地址', trigger: 'blur' }],
}

const addressPlaceholder = computed(() => {
  switch (form.protocol) {
    case 'opcua': return 'opc.tcp://127.0.0.1:4840'
    case 'modbus': return '127.0.0.1:502'
    case 'mqtt': return 'tcp://127.0.0.1:1883'
    case 's7': return '192.168.0.1:102'
    case 'bacnet': return '192.168.0.1:47808'
    default: return '设备地址'
  }
})

function openCreate() {
  isEdit.value = false
  editId.value = null
  Object.assign(form, defaultForm())
  visible.value = true
}

function openEdit(device: Device) {
  isEdit.value = true
  editId.value = device.id
  Object.assign(form, {
    name: device.name,
    protocol: device.protocol,
    address: device.address,
    config: device.config || '{}',
    description: device.description,
  })
  visible.value = true
}

function resetForm() {
  formRef.value?.resetFields()
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value && editId.value) {
      await updateDevice(editId.value, { ...form })
      ElMessage.success('设备已更新')
    } else {
      await createDevice({ ...form })
      ElMessage.success('设备已创建')
    }
    visible.value = false
    emit('saved')
  } finally {
    submitting.value = false
  }
}

defineExpose({ openCreate, openEdit })
</script>
