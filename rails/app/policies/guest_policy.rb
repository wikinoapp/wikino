# typed: strict
# frozen_string_literal: true

# 非ログイン・非スペースメンバー用の権限判定クラス。
# 公開トピック・ページの閲覧のみ許可し、それ以外はすべて拒否する。
class GuestPolicy
  extend T::Sig

  # トピック

  sig { returns(T::Boolean) }
  def can_update_topic?
    false
  end

  sig { returns(T::Boolean) }
  def can_delete_topic?
    false
  end

  # ページ

  sig { returns(T::Boolean) }
  def can_create_page?
    false
  end

  # 下書きページ

  sig { params(is_owner: T::Boolean).returns(T::Boolean) }
  def can_show_draft_page?(is_owner:) # rubocop:disable Lint/UnusedMethodArgument
    false
  end

  sig { params(is_owner: T::Boolean).returns(T::Boolean) }
  def can_update_draft_page?(is_owner:) # rubocop:disable Lint/UnusedMethodArgument
    false
  end

  # スペース

  sig { returns(T::Boolean) }
  def can_update_space?
    false
  end

  sig { returns(T::Boolean) }
  def can_export_space?
    false
  end

  # トピック作成

  sig { returns(T::Boolean) }
  def can_create_topic?
    false
  end

  # ゴミ箱

  sig { returns(T::Boolean) }
  def can_show_trash?
    false
  end

  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    false
  end

  # 添付ファイル

  sig { returns(T::Boolean) }
  def can_upload_attachment?
    false
  end

  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    attachment_record.all_referencing_pages_public?
  end

  sig { returns(T::Boolean) }
  def can_delete_attachment?
    false
  end
end
