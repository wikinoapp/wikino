# typed: strict
# frozen_string_literal: true

class EditPageForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

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
    page = topic&.pages&.find_by(title:)

    if page
      edit_page_path = "/s/#{topic.not_nil!.space.identifier}/pages/#{page.number}/edit"
      errors.add(:title, I18n.t("forms.errors.models.edit_page_form.uniqueness_html", edit_page_path:))
    end
  end
end
