# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRecord < ApplicationRecord
  self.table_name = "edit_suggestion_pages"

  belongs_to :space, class_name: "SpaceRecord"
  belongs_to :edit_suggestion, class_name: "EditSuggestionRecord"
  belongs_to :page, class_name: "PageRecord", optional: true

  sig { returns(T::Boolean) }
  def new_page?
    page_id.nil?
  end

  sig { returns(T::Boolean) }
  def existing_page?
    page_id.present?
  end

  sig { returns(T::Boolean) }
  def title_changed?
    title_before != title_after
  end

  sig { returns(T::Boolean) }
  def body_changed?
    body_before != body_after
  end

  sig { returns(T::Boolean) }
  def has_changes?
    title_changed? || body_changed?
  end
end
