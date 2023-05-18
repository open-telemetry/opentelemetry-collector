/**
 * Copyright Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


            const milestones = github.rest.issues.listMilestones({
              owner: context.repo.owner,
              repo: context.repo.repo,
              state: "open"
            })
            console.log(mielstones)

            for (const milestone of milestones) {
              console.log(milestone)
              if (milestone.title.startsWith("v")) {
                console.log("found it")
                //await github.issues.update({
                  //owner: context.repo.owner,
                  //repo: context.repo.repo,
                  //issue_number: ${{ github.event.pull_request.number }},
                  //milestone: milestone.number
                //});
                //return
              }
            }