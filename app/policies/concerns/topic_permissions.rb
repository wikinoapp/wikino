# typed: strict
# frozen_string_literal: true

module TopicPermissions
  extend T::Sig
  extend T::Helpers

  abstract!

  # Topic管理権限
  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_show_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
  end

  # Page操作権限
  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
  end

  # Draft操作権限
  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
  end

  # 編集提案権限
  sig { abstract.returns(T::Boolean) }
  def can_create_edit_suggestion?
  end

  sig { abstract.params(edit_suggestion_record: EditSuggestionRecord).returns(T::Boolean) }
  def can_update_edit_suggestion?(edit_suggestion_record:)
  end

  sig { abstract.params(edit_suggestion_record: EditSuggestionRecord).returns(T::Boolean) }
  def can_apply_edit_suggestion?(edit_suggestion_record:)
  end

  sig { abstract.params(edit_suggestion_record: EditSuggestionRecord).returns(T::Boolean) }
  def can_close_edit_suggestion?(edit_suggestion_record:)
  end

  sig { abstract.params(edit_suggestion_record: EditSuggestionRecord).returns(T::Boolean) }
  def can_comment_on_edit_suggestion?(edit_suggestion_record:)
  end
end
