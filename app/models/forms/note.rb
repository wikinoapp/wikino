# typed: strict
# frozen_string_literal: true

module Forms
  class Note < Forms::Base
    extend T::Sig

    sig { returns(T.nilable(User)) }
    attr_reader :user

    sig { returns(T.nilable(String)) }
    attr_reader :note_id

    validates :body, length: {maximum: 1_000_000}
    validates :title, presence: true
    validates :user, presence: true
    validate :title_should_be_unique

    sig { params(value: T.nilable(User)).void }
    def user=(value)
      @user = T.let(value, T.nilable(User))
    end

    sig { params(value: T.nilable(String)).void }
    def note_id=(value)
      @note_id = T.let(value, T.nilable(String))
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

    # @overload
    sig { returns(T::Boolean) }
    def persisted?
      note_id.present?
    end

    private

    sig { void }
    def title_should_be_unique
      return unless user

      notes = T.must(user).notes.where(title:)

      if note_id
        notes = notes.where.not(id: note_id)
      end

      if notes.exists?
        errors.add(:title, :title_should_be_unique)
      end
    end
  end
end
