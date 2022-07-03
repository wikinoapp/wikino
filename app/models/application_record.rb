# typed: strict
# frozen_string_literal: true

class ApplicationRecord < ActiveRecord::Base
  extend T::Sig

  self.abstract_class = true
end
