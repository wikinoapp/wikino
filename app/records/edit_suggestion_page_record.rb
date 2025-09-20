# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRecord < ApplicationRecord
  self.table_name = "edit_suggestion_pages"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_record, foreign_key: :edit_suggestion_id
  belongs_to :page_record, foreign_key: :page_id, optional: true
  belongs_to :page_revision_record, foreign_key: :page_revision_id, optional: true

  sig { returns(T::Boolean) }
  def new_page?
    page_id.nil? && page_revision_id.nil?
  end

  sig { returns(T::Boolean) }
  def existing_page?
    !new_page?
  end

  sig { returns(T::Boolean) }
  def title_changed?
    return true if new_page?

    page_revision_record.not_nil!.title != title
  end

  sig { returns(T::Boolean) }
  def body_changed?
    return true if new_page?

    page_revision_record.not_nil!.body != body
  end

  sig { returns(T::Boolean) }
  def has_changes?
    title_changed? || body_changed?
  end
end
