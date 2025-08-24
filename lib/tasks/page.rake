# typed: false
# frozen_string_literal: true

namespace :page do
  desc "ゴミ箱に入っているページを削除する"
  task bulk_destroy_trashed: :environment do
    Pages::BulkDestroyTrashedService.new.call
  end
end
