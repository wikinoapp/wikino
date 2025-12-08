# typed: true
# frozen_string_literal: true

module ModelConcerns
  module RecordError
    extend ActiveSupport::Concern
    extend T::Sig

    included do
      validate :check_record_errors
    end

    sig { params(args: T.untyped).returns(T::Boolean) }
    def add_record_error(*args)
      @static_errors ||= []
      @static_errors << args

      true
    end

    sig { void }
    def clear_record_errors
      @static_errors = nil
    end

    sig { void }
    private def check_record_errors
      @static_errors&.each do |error|
        errors.add(*error)
      end
    end
  end
end
