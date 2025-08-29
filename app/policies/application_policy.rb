# typed: strict
# frozen_string_literal: true

# 全てのPolicyクラスの基底クラス
class ApplicationPolicy
  extend T::Sig
  extend T::Helpers
  abstract!

  sig { params(user_record: T.nilable(UserRecord)).void }
  def initialize(user_record:)
    @user_record = user_record
  end

  # 抽象メソッド - 全ての子クラスで実装が必要
  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
  end

  sig { abstract.returns(T::Boolean) }
  def can_create_topic?
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
  end

  sig { abstract.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
  end

  sig { abstract.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
  end

  sig { abstract.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
  end

  sig { abstract.returns(T::Boolean) }
  def joined_space?
  end

  sig { abstract.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
  end

  sig { abstract.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record
end
