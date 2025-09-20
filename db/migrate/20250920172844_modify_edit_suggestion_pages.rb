# typed: false
# frozen_string_literal: true

class ModifyEditSuggestionPages < ActiveRecord::Migration[8.0]
  disable_ddl_transaction!

  def change
    # 最新リビジョンへの参照を追加
    add_reference :edit_suggestion_pages, :latest_revision, type: :uuid, index: {algorithm: :concurrently}

    # title と body カラムを削除
    safety_assured do
      remove_column :edit_suggestion_pages, :title, :citext
      remove_column :edit_suggestion_pages, :body, :citext
    end
  end
end
