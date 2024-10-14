# typed: strict
# frozen_string_literal: true

class EditPageForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

  sig { returns(T.nilable(Page)) }
  attr_accessor :page

  attribute :topic_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :topic, presence: true
  validates :title, presence: true
  validates :body, presence: true, allow_blank: true
  validate :title_uniqueness

  sig { returns(T.nilable(Topic)) }
  def topic
    viewer&.viewable_topics&.find_by(number: topic_number)
  end

  sig { returns(Topic::PrivateRelation) }
  def viewable_topics
    viewer.not_nil!.viewable_topics
  end

  sig { returns(T::Boolean) }
  def autofocus_title?
    title.blank?
  end

  sig { returns(T::Boolean) }
  def autofocus_body?
    !autofocus_title?
  end

  sig { void }
  private def title_uniqueness
    return if topic.nil?

    other_page = topic.not_nil!.pages.where.not(id: page.not_nil!.id).find_by(title:)

    if other_page
      edit_page_path = "/s/#{topic.not_nil!.space.not_nil!.identifier}/pages/#{other_page.number}/edit"
      errors.add(:title, I18n.t("forms.errors.models.edit_page_form.uniqueness_html", edit_page_path:))
    end
  end
end
