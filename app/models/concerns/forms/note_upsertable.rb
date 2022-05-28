# typed: strict
# frozen_string_literal: true

require "github/markup"

module Forms
  module NoteUpsertable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      validates :user, presence: true
      validates :body, length: {maximum: 1_000_000}
    end

    sig { returns(T.nilable(User)) }
    attr_reader :user

    sig { returns(T.nilable(::Note)) }
    attr_accessor :original_note

    sig { params(value: T.nilable(User)).void }
    def user=(value)
      @user = T.let(value, T.nilable(User))
    end

    sig { params(value: T.nilable(String)).void }
    def title=(value)
      @title = T.let(value, T.nilable(String))
    end

    sig { params(value: T.nilable(String)).void }
    def body=(value)
      @body = T.let(value, T.nilable(String))
    end

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

    sig { params(body: String).returns(String) }
    private def render_html(body)
      GitHub::Markup.render_s(
        GitHub::Markups::MARKUP_MARKDOWN,
        body,
        options: { commonmarker_opts: %i(HARDBREAKS) }
      )
    end
  end
end
