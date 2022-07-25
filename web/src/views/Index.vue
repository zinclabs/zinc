<template>
  <q-page class="q-pa-md">
    <q-table
      v-model:selected="selectedIndexes"
      v-model:pagination="pagination"
      :title="t('index.header')"
      :rows="indexes"
      :columns="resultColumns"
      row-key="name"
      selection="multiple"
      :loading="loading"
      :filter="filterQuery"
      :rows-per-page-options="rowsPerPageOptions"
      @request="onRequest"
    >
      <template #top-right>
        <q-input
          v-model="filterQuery"
          filled
          borderless
          dense
          :placeholder="t('index.search')"
        >
          <template #append>
            <q-icon name="search" class="cursor-pointer" @click="getIndexes" />
          </template>
        </q-input>
        <q-btn
          class="q-ml-sm"
          color="primary"
          icon="add"
          :label="t('index.add')"
          @click="addIndex"
        />
        <q-btn
          class="q-ml-sm"
          color="negative"
          icon="delete"
          :label="t('index.delete')"
          @click="deleteSelectedIndexes"
        />
      </template>

      <template #body-cell-no="props">
        <q-td :props="props" width="80">
          {{ props.value }}
        </q-td>
      </template>
      <template #body-cell-name="props">
        <q-td :props="props" auto-width>
          <a
            class="text-primary text-decoration-none"
            @click="previewIndex(props)"
          >
            {{ props.value }}
          </a>
        </q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" auto-width>
          <q-btn
            dense
            unelevated
            size="sm"
            color="blue-5"
            class="action-button"
            icon="description"
            @click="previewIndex(props)"
          />
          <q-btn
            dense
            unelevated
            size="sm"
            color="red-5"
            class="action-button q-ml-sm"
            icon="delete"
            @click="deleteIndex(props)"
          />
        </q-td>
      </template>
    </q-table>

    <q-dialog
      v-model="showAddIndexDialog"
      position="right"
      full-height
      seamless
      maximized
    >
      <add-update-index @updated="indexAdded" />
    </q-dialog>

    <q-dialog
      v-model="showPreviewIndexDialog"
      position="right"
      full-height
      maximized
    >
      <preview-index v-model="index" />
    </q-dialog>
  </q-page>
</template>

<script>
import { defineComponent, nextTick, ref } from "vue";
import { useStore } from "vuex";
import { useQuasar } from "quasar";
import { useI18n } from "vue-i18n";

import indexService from "../services/index";
import AddUpdateIndex from "../components/index/AddUpdateIndex.vue";
import PreviewIndex from "../components/index/PreviewIndex.vue";

export default defineComponent({
  name: "PageIndex",
  components: {
    AddUpdateIndex,
    PreviewIndex,
  },
  setup() {
    const store = useStore();
    const $q = useQuasar();
    const { t } = useI18n();
    const loading = ref(false);
    const selectedIndexes = ref([]);
    const filterQuery = ref("");
    const indexes = ref([]);
    const rowsPerPageOptions = ref([5, 10, 20, 50, 100, 500, 1000]);
    const pagination = ref({
      rowsPerPage: 20,
      sortBy: "name",
      descending: false,
      page: 1,
      rowsNumber: 0,
    });
    const onRequest = (props) => {
      const { page, rowsPerPage, sortBy, descending } = props.pagination;
      pagination.value.page = page;
      pagination.value.rowsPerPage = rowsPerPage;
      pagination.value.sortBy = sortBy;
      pagination.value.descending = descending;
      getIndexes();
    };
    const getIndexes = () => {
      loading.value = true;
      let page_num = pagination.value.page;
      let page_size = pagination.value.rowsPerPage;
      indexService
        .list(
          page_num,
          page_size,
          pagination.value.sortBy,
          pagination.value.descending,
          filterQuery.value
        )
        .then((res) => {
          var counter = 1;
          pagination.value.rowsNumber = res.data.page.total;
          indexes.value = res.data.list.map((data) => {
            let storage_size =
              (data.stats.storage_size / 1024).toFixed(2) + " KB";
            if (data.stats.storage_size > 1024 * 1024) {
              storage_size =
                (data.stats.storage_size / 1024 / 1024).toFixed(2) + " MB";
            }
            if (data.stats.storage_size > 1024 * 1024 * 1024) {
              storage_size =
                (data.stats.storage_size / 1024 / 1024 / 1024).toFixed(2) +
                " GB";
            }
            return {
              no: counter++,
              name: data.name,
              doc_num: data.stats.doc_num,
              shard_num: data.shard_num,
              storage_size: storage_size,
              storage_type: data.storage_type,
              wal_size: data.stats.wal_size,
              actions: {
                settings: data.settings,
                mappings: data.mappings,
              },
            };
          });
          loading.value = false;
        });
    };

    getIndexes();

    const resultColumns = [
      {
        name: "no",
        field: (row) => row.no,
        label: "#",
        align: "right",
        sortable: true,
      },
      {
        name: "name",
        field: (row) => row.name,
        label: "NAME",
        align: "left",
        sortable: true,
      },
      {
        name: "doc_num",
        field: (row) => row.doc_num,
        label: "DOC_NUM",
        align: "right",
        sortable: true,
      },
      {
        name: "shard_num",
        field: (row) => row.shard_num,
        label: "SHARD_NUM",
        align: "right",
        sortable: true,
      },
      {
        name: "storage_size",
        field: (row) => row.storage_size,
        label: "STORAGE_SIZE",
        align: "right",
        sortable: true,
      },
      {
        name: "storage_type",
        field: (row) => row.storage_type,
        label: "STORAGE_TYPE",
        align: "left",
        sortable: true,
      },
      {
        name: "actions",
        field: (row) => row.actions,
        label: "ACTIONS",
        align: "left",
        sortable: true,
      },
    ];

    const index = ref({});
    const showAddIndexDialog = ref(false);
    const showPreviewIndexDialog = ref(false);

    const addIndex = () => {
      showAddIndexDialog.value = true;
    };
    const previewIndex = (props) => {
      index.value = {
        name: props.row.name,
        doc_num: props.row.doc_num,
        shard_num: props.row.shard_num,
        storage_size: props.row.storage_size,
        storage_type: props.row.storage_type,
        wal_size: props.row.wal_size,
        settings: props.row.actions.settings,
        mappings: props.row.actions.mappings,
      };
      showPreviewIndexDialog.value = true;
    };
    const deleteIndex = (props) => {
      $q.dialog({
        title: "Delete index",
        message:
          "You are about to delete this index: <ul><li>" +
          props.row.name +
          "</li></ul>",
        cancel: true,
        persistent: true,
        html: true,
      }).onOk(() => {
        indexService.delete(props.row.name).then(() => {
          nextTick(getIndexes);
        });
      });
    };

    const deleteSelectedIndexes = () => {
      if (!selectedIndexes.value || selectedIndexes.value.length == 0) {
        $q.notify({
          position: "top",
          color: "warning",
          textColor: "white",
          icon: "warning",
          message: "Please select index for deletion",
        });
        return;
      }

      const showText = selectedIndexes.value
        .map((r) => "<li>" + r.name + "</li>")
        .join("");
      $q.dialog({
        title: "Delete indexes",
        message:
          "You are about to delete these indexes: <ul>" + showText + "</ul>",
        cancel: true,
        persistent: true,
        html: true,
      }).onOk(() => {
        const indexNames = selectedIndexes.value.map((r) => r.name).join(",");
        loading.value = true;
        indexService.delete(indexNames).then((res) => {
          loading.value = false;
          selectedIndexes.value = [];
          nextTick(getIndexes);
        });
      });
    };

    return {
      t,
      showAddIndexDialog,
      showPreviewIndexDialog,
      resultColumns,
      index,
      loading,
      selectedIndexes,
      deleteSelectedIndexes,
      indexes,
      rowsPerPageOptions,
      pagination,
      onRequest,
      filterQuery,
      getIndexes,
      addIndex,
      deleteIndex,
      previewIndex,
      indexAdded() {
        showAddIndexDialog.value = false;
        getIndexes();
      },
    };
  },
});
</script>
