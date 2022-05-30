# typed: strict
# frozen_string_literal: true

require "github/markup"

module NoteInputtable
  extend T::Sig
  extend ActiveSupport::Concern
  include Kernel

  included do
    validates :user, presence: true
    validates :body, length: {maximum: 1_000_000}
    validate :title_should_be_unique
  end

  sig { returns(T.nilable(User)) }
  attr_accessor :user

  sig { params(title: T.nilable(String)).returns(T.nilable(String)) }
  attr_writer :title

  sig { params(body: T.nilable(String)).returns(T.nilable(String)) }
  attr_writer :body

  sig { returns(T.nilable(::Note)) }
  attr_accessor :original_note

  sig { returns(String) }
  def title
    @title.presence || "No Title @#{Time.current.to_i}"
  end

  sig { returns(String) }
  def body
    @body || ""
  end

  sig { returns(String) }
  def body_html
    render_html(body)
  end

  private

  sig { returns(ActiveRecord::Relation) }
  def user_notes
    raise NotImplementedError
  end

  sig { void }
  def title_should_be_unique
    return unless user

    @original_note = user_notes.where(title:).first

    if @original_note
      errors.add(:title, :title_should_be_unique)
    end
  end

  sig { params(body: String).returns(String) }
  def render_html(body)
    GitHub::Markup.render_s(
      GitHub::Markups::MARKUP_MARKDOWN,
      body,
      options: {commonmarker_opts: %i[HARDBREAKS]}
    )
  end
end
