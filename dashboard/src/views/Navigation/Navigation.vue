<script lang="ts" src="./Navigation.ts"></script>

<template>
      <div class="create_task" :class="{
      headerIsFixed: fixedStatus.headerIsFixed
    }">
      <el-card>
        <h3 class="heading">vue-fixed-header debugging</h3>
        <el-form label-width="120px">
          <el-form-item label="threshold">
            <el-input type="number" v-model="formData.threshold"></el-input>
          </el-form-item>
          <el-form-item label="headerClass">
            <el-input type="text" v-model="formData.headerClass"></el-input>
          </el-form-item>
          <el-form-item label="fixedClass">
            <el-input type="text" v-model="formData.fixedClass"></el-input>
          </el-form-item>
          <el-form-item label="hideScrollUp">
            <el-checkbox v-model="formData.hideScrollUp">hideScrollUp</el-checkbox>
          </el-form-item>
          <el-form-item>
            <div class="justify-right">
              <el-button type="primary" @click="handleClickUpdate">Update</el-button>
              <el-button type="secondary" @click="handleClickCancel">Cancel</el-button>
            </div>
          </el-form-item>
        </el-form>
      </el-card>
    </div>
</template>

<style lang="scss" src="./Navigation.scss" scoped></style>
