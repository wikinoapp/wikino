# typed: strict
# frozen_string_literal: true

module SpacePermissions
  extend T::Sig
  extend T::Helpers

  # SpacePermissionsはSpace関連の権限メソッドを提供
  # TopicAdminPolicyなどSpace権限を持たないクラスはこのモジュールをincludeしない

  # Space管理権限
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    raise NotImplementedError, "#{self.class}#can_update_space? must be implemented"
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:)
    raise NotImplementedError, "#{self.class}#can_manage_space_members? must be implemented"
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    raise NotImplementedError, "#{self.class}#can_export_space? must be implemented"
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:)
    raise NotImplementedError, "#{self.class}#can_delete_space? must be implemented"
  end

  # 添付ファイル管理権限（Spaceレベル）
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    raise NotImplementedError, "#{self.class}#can_manage_attachments? must be implemented"
  end

  # 添付ファイルアップロード権限（Spaceレベル）
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    raise NotImplementedError, "#{self.class}#can_upload_attachment? must be implemented"
  end

  # Space参加状態の確認
  sig { returns(T::Boolean) }
  def joined_space?
    raise NotImplementedError, "#{self.class}#joined_space? must be implemented"
  end

  # 参加しているトピック一覧
  sig { returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    raise NotImplementedError, "#{self.class}#joined_topic_records must be implemented"
  end

  # 閲覧可能なトピック一覧
  sig { params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    raise NotImplementedError, "#{self.class}#showable_topics must be implemented"
  end

  # 閲覧可能なページ一覧
  sig { params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    raise NotImplementedError, "#{self.class}#showable_pages must be implemented"
  end

  # トピック作成権限
  sig { returns(T::Boolean) }
  def can_create_topic?
    raise NotImplementedError, "#{self.class}#can_create_topic? must be implemented"
  end

  # ゴミ箱閲覧権限
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    raise NotImplementedError, "#{self.class}#can_show_trash? must be implemented"
  end

  # 一括復元権限
  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    raise NotImplementedError, "#{self.class}#can_create_bulk_restore_pages? must be implemented"
  end
end
