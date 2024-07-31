# typed: false
# frozen_string_literal: true

class Team < ApplicationRecord
  extend T::Sig

  include Discard::Model

  has_many :notes, dependent: :restrict_with_exception
end
