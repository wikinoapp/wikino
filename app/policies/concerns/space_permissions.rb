# typed: strict
# frozen_string_literal: true

module SpacePermissions
  extend T::Sig
  extend T::Helpers

  abstract!

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:); end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:); end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:); end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:); end

  # 添付ファイル管理権限（Spaceレベル）
  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:); end

  # 添付ファイルアップロード権限（Spaceレベル）
  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:); end
end