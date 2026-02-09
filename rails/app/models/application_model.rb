# typed: strict
# frozen_string_literal: true

class ApplicationModel
  extend T::Sig

  include ActiveModel::Model

  include ModelConcerns::RecordError
end
