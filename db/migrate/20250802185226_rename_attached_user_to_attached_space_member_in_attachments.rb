# typed: false
# frozen_string_literal: true

class RenameAttachedUserToAttachedSpaceMemberInAttachments < ActiveRecord::Migration[8.0]
  def change
    safety_assured do
      # カラムのリネーム
      rename_column :attachments, :attached_user_id, :attached_space_member_id

      # インデックスの削除（古い名前で存在する場合）
      if index_exists?(:attachments, :attached_user_id, name: "index_attachments_on_attached_user_id")
        remove_index :attachments, :attached_user_id, name: "index_attachments_on_attached_user_id"
      end

      # インデックスの追加（新しい名前で存在しない場合）
      unless index_exists?(:attachments, :attached_space_member_id, name: "index_attachments_on_attached_space_member_id")
        add_index :attachments, :attached_space_member_id, name: "index_attachments_on_attached_space_member_id"
      end

      # 外部キー制約を削除して再作成
      if foreign_key_exists?(:attachments, :users, column: :attached_user_id)
        remove_foreign_key :attachments, :users, column: :attached_user_id
      elsif foreign_key_exists?(:attachments, :users, column: :attached_space_member_id)
        remove_foreign_key :attachments, :users, column: :attached_space_member_id
      end

      add_foreign_key :attachments, :space_members, column: :attached_space_member_id
    end
  end
end
