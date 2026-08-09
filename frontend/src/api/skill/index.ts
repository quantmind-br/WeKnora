import { get } from "../../utils/request";

// Skill info
export interface SkillInfo {
  name: string;
  description: string;
}

// Get the list of pre-installed Skills; skills_available false means the sandbox isn't enabled, the frontend should hide/disable Skills configuration
export function listSkills() {
  return get<{ data: SkillInfo[]; skills_available?: boolean }>('/api/v1/skills');
}
